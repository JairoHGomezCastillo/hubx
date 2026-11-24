package shipment

import (
	"encoding/json"
	"errors"
	"net/http"

	domainshipment "hubx-api/internal/shipment"
)

// Transport-level sentinels. Domain sentinel errors stay in the domain package
// and are translated here, as required by the architecture.
var (
	// ErrSessionExpired means no valid principal was supplied by the middleware.
	ErrSessionExpired = errors.New("session expired; sign in again")
	// ErrInvalidRequest means the request body is malformed or has an invalid stage.
	ErrInvalidRequest = errors.New("invalid update request")
)

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type updateErrorResponse struct {
	Error errorDetail `json:"error"`
}

// writeUpdateError maps a domain or transport error to the contract's JSON
// error body. Unknown and invisible shipment codes arrive as ErrNotResponsible,
// so no response reveals whether a code exists (FR-012).
func writeUpdateError(w http.ResponseWriter, err error) {
	status, code, message := httpError(err)
	body, marshalErr := json.Marshal(updateErrorResponse{
		Error: errorDetail{Code: code, Message: message},
	})
	if marshalErr != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func httpError(err error) (int, string, string) {
	switch {
	case errors.Is(err, ErrSessionExpired):
		return http.StatusUnauthorized, "SESSION_EXPIRED", "Session expired; sign in again."
	case errors.Is(err, ErrInvalidRequest):
		return http.StatusBadRequest, "INVALID_REQUEST", "The update data is invalid."
	case errors.Is(err, domainshipment.ErrPartnerRequired):
		return http.StatusBadRequest, "PARTNER_REQUIRED", "You must name the ground partner responsible for the delivery."
	case errors.Is(err, domainshipment.ErrNotGroundPartner):
		return http.StatusForbidden, "NOT_GROUND_PARTNER", "Only a ground partner can be responsible for the delivery."
	case errors.Is(err, domainshipment.ErrNotResponsible):
		return http.StatusForbidden, "NOT_RESPONSIBLE", "You are not responsible for that update."
	case errors.Is(err, domainshipment.ErrOutOfSequence):
		return http.StatusConflict, "OUT_OF_SEQUENCE", "That update does not follow the shipment's sequence."
	case errors.Is(err, domainshipment.ErrConcurrentUpdate):
		return http.StatusConflict, "CONCURRENT_UPDATE", "The shipment changed while recording the update; try again."
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR", "The update could not be recorded."
	}
}

func writeUpdateJSON(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		writeUpdateError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
