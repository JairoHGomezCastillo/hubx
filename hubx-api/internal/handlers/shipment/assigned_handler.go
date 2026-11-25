package shipment

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	domainshipment "hubx-api/internal/shipment"
)

//go:generate go tool mockgen -source=assigned_handler.go -destination=mocks/assigned_handler_mock.go -package=mocks

// AssignedLister lists shipments assigned to a partner that are not yet delivered.
type AssignedLister interface {
	List(ctx context.Context, principal domainshipment.Principal, offset int) (domainshipment.AssignedPage, error)
}

type assignedShipmentResponse struct {
	Code        string `json:"code"`
	Stage       string `json:"stage"`
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	Flight      string `json:"flight"`
	WeightKg    any    `json:"weightKg"`
	AssignedAt  string `json:"assignedAt"`
}

type assignedResponse struct {
	Shipments  []assignedShipmentResponse `json:"shipments"`
	NextOffset *int                       `json:"nextOffset"`
	Message    *string                    `json:"message"`
}

type assignedHandler struct {
	lister     AssignedLister
	principals PrincipalSource
}

// RegisterAssigned mounts the HU-07 assigned shipments route on mux. It uses the
// flat 401 body from RequireAuth (not the nested SESSION_EXPIRED body), keeping
// it consistent with HU-01, HU-02 and HU-04.
func RegisterAssigned(
	mux *http.ServeMux,
	requireAuth func(http.Handler) http.Handler,
	lister AssignedLister,
	principals PrincipalSource,
) {
	h := &assignedHandler{lister: lister, principals: principals}
	mux.Handle("GET /assigned-shipments", requireAuth(http.HandlerFunc(h.handleAssigned)))
}

func (h *assignedHandler) handleAssigned(w http.ResponseWriter, r *http.Request) {
	principal, err := h.principals.Principal(r)
	if err != nil || principal.UserID == "" || principal.Role == "" {
		writeAssignedError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid authorization.")
		return
	}

	// Parse offset: digits only, default 0, invalid becomes -1 so domain checks role first
	offsetStr := r.URL.Query().Get("offset")
	offset := 0
	if offsetStr != "" {
		if !isDigits(offsetStr) {
			offset = -1
		} else {
			parsed, err := strconv.Atoi(offsetStr)
			if err != nil {
				offset = -1
			} else {
				offset = parsed
			}
		}
	}

	page, err := h.lister.List(r.Context(), principal, offset)
	if err != nil {
		if errors.Is(err, domainshipment.ErrNoPermission) {
			writeAssignedError(w, http.StatusForbidden, "FORBIDDEN", "You do not have permission to see that information")
		} else if errors.Is(err, domainshipment.ErrInvalidOffset) {
			writeAssignedError(w, http.StatusBadRequest, "INVALID_REQUEST", "The offset must be a whole number between 0 and 100000")
		} else {
			slog.Error("assigned shipments list failed", "error", err)
			writeAssignedError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
		}
		return
	}

	// Project domain page onto HTTP contract
	shipments := make([]assignedShipmentResponse, len(page.Shipments))
	for i, s := range page.Shipments {
		shipments[i] = assignedShipmentResponse{
			Code:        s.Code,
			Stage:       string(s.Stage),
			Origin:      s.Origin,
			Destination: s.Destination,
			Flight:      s.Flight,
			WeightKg:    parseWeightJSON(s.WeightKg),
			AssignedAt:  s.AssignedAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"),
		}
	}

	response := assignedResponse{
		Shipments:  shipments,
		NextOffset: page.NextOffset,
		Message:    page.Message,
	}

	writeAssignedJSON(w, http.StatusOK, response)
}

func writeAssignedError(w http.ResponseWriter, status int, code, message string) {
	body, err := json.Marshal(map[string]string{
		"error":   code,
		"message": message,
	})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func writeAssignedJSON(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		writeAssignedError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// parseWeightJSON converts the domain's decimal literal to a JSON number for the contract.
// HU-02 stores it as a string; the HU-07 contract returns it as a JSON number.
func parseWeightJSON(weightKg string) any {
	// Weight is stored as a validated literal; a parse error should not happen.
	// Return the string as fallback so the response stays valid.
	f, err := strconv.ParseFloat(weightKg, 64)
	if err != nil {
		return weightKg
	}
	return f
}
