package shipment

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	domainshipment "hubx-api/internal/shipment"
)

//go:generate go tool mockgen -source=traceability_handler.go -destination=mocks/traceability_handler_mock.go -package=mocks
//go:generate go tool mockgen -source=../../shipment/traceability.go -destination=mocks/traceability_reader_mock.go -package=mocks

type traceabilityResponse struct {
	ShipmentCode string               `json:"shipmentCode"`
	Stage        string               `json:"stage"`
	Updates      []traceabilityUpdate `json:"updates"`
}

type traceabilityUpdate struct {
	Sequence            int              `json:"sequence"`
	Stage               string           `json:"stage"`
	ResponsibleParty    responsibleParty `json:"responsibleParty"`
	AssignedPartner     *party           `json:"assignedPartner"`
	RecordedAt          string           `json:"recordedAt"`
	PreviousFingerprint *string          `json:"previousFingerprint"`
	Fingerprint         string           `json:"fingerprint"`
}

type traceabilityHandler struct {
	reader     domainshipment.TraceabilityReader
	principals PrincipalSource
}

// RegisterTraceability mounts the HU-04 traceability route on mux. It uses the
// flat 401 body from RequireAuth (not the nested SESSION_EXPIRED body), keeping
// it consistent with HU-01 and HU-02.
func RegisterTraceability(
	mux *http.ServeMux,
	requireAuth func(http.Handler) http.Handler,
	reader domainshipment.TraceabilityReader,
	principals PrincipalSource,
) {
	h := &traceabilityHandler{reader: reader, principals: principals}
	mux.Handle("GET /shipments/{code}/traceability", requireAuth(http.HandlerFunc(h.handleTraceability)))
}

func (h *traceabilityHandler) handleTraceability(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	principal, err := h.principals.Principal(r)
	if err != nil || principal.UserID == "" || principal.Role == "" {
		writeTraceabilityError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid authorization.")
		return
	}

	traceability, err := h.reader.Read(r.Context(), code, principal)
	if err != nil {
		if errors.Is(err, domainshipment.ErrShipmentNotFound) {
			writeTraceabilityError(w, http.StatusNotFound, "SHIPMENT_NOT_FOUND", "No shipment exists with that code.")
		} else {
			slog.Error("traceability read failed", "error", err)
			writeTraceabilityError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "The traceability could not be retrieved.")
		}
		return
	}

	// Project domain view onto HTTP contract
	updates := make([]traceabilityUpdate, len(traceability.Updates))
	for i, view := range traceability.Updates {
		update := traceabilityUpdate{
			Sequence: view.Sequence,
			Stage:    view.Stage,
			ResponsibleParty: responsibleParty{
				ID:   view.ResponsibleParty.ID,
				Name: view.ResponsibleParty.Name,
				Role: view.ResponsibleParty.Role,
			},
			RecordedAt:          view.RecordedAt,
			PreviousFingerprint: view.PreviousFingerprint,
			Fingerprint:         view.Fingerprint,
		}
		if view.AssignedPartner != nil {
			update.AssignedPartner = &party{
				ID:   view.AssignedPartner.ID,
				Name: view.AssignedPartner.Name,
			}
		}
		updates[i] = update
	}

	response := traceabilityResponse{
		ShipmentCode: traceability.ShipmentCode,
		Stage:        traceability.Stage,
		Updates:      updates,
	}

	writeTraceabilityJSON(w, http.StatusOK, response)
}

func writeTraceabilityError(w http.ResponseWriter, status int, code, message string) {
	var body []byte
	var err error

	if status == http.StatusUnauthorized {
		// Flat body for 401, consistent with RequireAuth (HU-01)
		body, err = json.Marshal(map[string]any{
			"error":   code,
			"message": message,
		})
	} else {
		// Nested body for 404 and 500, consistent with HU-03
		body, err = json.Marshal(map[string]any{
			"error": map[string]string{
				"code":    code,
				"message": message,
			},
		})
	}

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func writeTraceabilityJSON(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		writeTraceabilityError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "The traceability could not be retrieved.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
