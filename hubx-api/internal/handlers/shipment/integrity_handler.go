package shipment

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	domainshipment "hubx-api/internal/shipment"
)

//go:generate go tool mockgen -source=../../shipment/verifier.go -destination=mocks/verifier_mock.go -package=mocks

// Verifier verifies the integrity of a shipment's update chain.
type Verifier interface {
	Verify(ctx context.Context, code string, principal domainshipment.Principal) (*domainshipment.Verification, error)
}

type integrityResponse struct {
	ShipmentCode       string `json:"shipmentCode"`
	Intact             bool   `json:"intact"`
	UpdatesChecked     int    `json:"updatesChecked"`
	FirstAlteredUpdate *int   `json:"firstAlteredUpdate"` // null when intact
}

type integrityHandler struct {
	verifier   Verifier
	principals PrincipalSource
}

// RegisterIntegrity mounts the HU-05 integrity verification route on mux.
// It goes through the HU-01 flat-body RequireAuth middleware.
func RegisterIntegrity(
	mux *http.ServeMux,
	requireAuth func(http.Handler) http.Handler,
	verifier Verifier,
) {
	h := &integrityHandler{verifier: verifier, principals: NewClaimsPrincipalSource()}
	mux.Handle("GET /shipments/{code}/integrity", requireAuth(http.HandlerFunc(h.handleIntegrity)))
}

func (h *integrityHandler) handleIntegrity(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	// Extract principal from context (set by RequireAuth)
	principal, err := h.principals.Principal(r)
	if err != nil || principal.UserID == "" || principal.Role == "" {
		writeIntegrityError(w, err)
		return
	}

	verification, err := h.verifier.Verify(r.Context(), code, principal)
	if err != nil {
		writeIntegrityError(w, err)
		return
	}

	response := integrityResponse{
		ShipmentCode:   verification.ShipmentCode,
		Intact:         verification.Intact,
		UpdatesChecked: verification.UpdatesChecked,
	}
	if !verification.Intact {
		response.FirstAlteredUpdate = &verification.FirstAlteredUpdate
	}

	writeIntegrityJSON(w, http.StatusOK, response)
}

type integrityErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// writeIntegrityError maps domain errors to the HU-05 flat error body
// (same shape as HU-01/HU-02).
func writeIntegrityError(w http.ResponseWriter, err error) {
	status, code, message := integrityHTTPError(err)
	body, marshalErr := json.Marshal(integrityErrorResponse{
		Error:   code,
		Message: message,
	})
	if marshalErr != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func integrityHTTPError(err error) (int, string, string) {
	switch {
	case errors.Is(err, domainshipment.ErrShipmentNotFound):
		return http.StatusNotFound, "SHIPMENT_NOT_FOUND", "No shipment exists with that code"
	default:
		return http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred"
	}
}

func writeIntegrityJSON(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		writeIntegrityError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
