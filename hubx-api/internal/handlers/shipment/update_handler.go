// Package shipment exposes the HTTP transport for recording shipment updates.
// It owns no business rules: validation of the JWT and of the gateway shared
// secret belongs to the HU-01 middleware, which supplies the authenticated
// party through PrincipalSource.
package shipment

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"hubx-api/internal/handlers/auth"
	"hubx-api/internal/ledger"
	domainshipment "hubx-api/internal/shipment"
)

//go:generate go tool mockgen -source=update_handler.go -destination=mocks/handler_mock.go -package=mocks
//go:generate go tool mockgen -source=../../shipment/recorder.go -destination=mocks/recorder_mock.go -package=mocks
//go:generate go tool mockgen -source=../../shipment/repository.go -destination=mocks/repository_mock.go -package=mocks

// PrincipalSource returns the signed-in party identified by the HU-01
// middleware. An error means the session is missing, invalid or expired.
type PrincipalSource interface {
	Principal(r *http.Request) (domainshipment.Principal, error)
}

// PartyDirectory resolves the display name of a party from its identifier.
// The SQL implementation is wired by cmd/api after the transport is ready;
// the handler never hard-codes user names.
type PartyDirectory interface {
	Name(ctx context.Context, userID string) (string, error)
}

// recordUpdateRequest is the strict JSON body of
// POST /shipments/{code}/updates.
type recordUpdateRequest struct {
	Stage             string  `json:"stage"`
	AssignedPartnerID *string `json:"assignedPartnerId"`
}

type party struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type responsibleParty struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type updateResponse struct {
	ID                  int64            `json:"id"`
	ShipmentCode        string           `json:"shipmentCode"`
	Sequence            int              `json:"sequence"`
	Stage               string           `json:"stage"`
	ResponsibleParty    responsibleParty `json:"responsibleParty"`
	AssignedPartner     *party           `json:"assignedPartner"`
	RecordedAt          time.Time        `json:"recordedAt"`
	PreviousFingerprint *string          `json:"previousFingerprint"`
	Fingerprint         string           `json:"fingerprint"`
}

type handler struct {
	recorder   domainshipment.Recorder
	principals PrincipalSource
	directory  PartyDirectory
	partners   domainshipment.GroundPartnerRepository
}

// RegisterUpdates mounts the HU-03 update routes on mux with Go 1.22+ method
// patterns. It is named separately from the HU-02 shipment creation Register
// so both stories can be wired on the same mux. Every route goes through the
// injected authentication middleware: the API validates the JWT on its own,
// and cmd/api injects the HU-03 session-error variant so rejected sessions get
// the nested SESSION_EXPIRED body.
func RegisterUpdates(
	mux *http.ServeMux,
	requireAuth func(http.Handler) http.Handler,
	recorder domainshipment.Recorder,
	principals PrincipalSource,
	directory PartyDirectory,
	partners domainshipment.GroundPartnerRepository,
) {
	h := &handler{recorder: recorder, principals: principals, directory: directory, partners: partners}
	mux.Handle("POST /shipments/{code}/updates", requireAuth(http.HandlerFunc(h.handleUpdate)))
	mux.Handle("GET /ground-partners", requireAuth(http.HandlerFunc(h.handleGroundPartners)))
}

// claimsPrincipalSource adapts the HU-01 JWT middleware to the transport's
// PrincipalSource interface without coupling the domain to net/http.
type claimsPrincipalSource struct{}

// NewClaimsPrincipalSource returns the JWT-backed PrincipalSource used by the
// wiring layer.
func NewClaimsPrincipalSource() PrincipalSource {
	return claimsPrincipalSource{}
}

func (claimsPrincipalSource) Principal(r *http.Request) (domainshipment.Principal, error) {
	claims := auth.GetClaims(r.Context())
	if claims == nil || claims.Sub == "" || claims.Role == "" {
		return domainshipment.Principal{}, ErrSessionExpired
	}
	return domainshipment.Principal{UserID: claims.Sub, Role: domainshipment.Role(claims.Role)}, nil
}

func (h *handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	principal, err := h.principals.Principal(r)
	if err != nil || principal.UserID == "" || principal.Role == "" {
		writeUpdateError(w, ErrSessionExpired)
		return
	}

	var body recordUpdateRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		writeUpdateError(w, ErrInvalidRequest)
		return
	}
	// A second decode must find only the end of the input: one JSON object.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeUpdateError(w, ErrInvalidRequest)
		return
	}

	stage := domainshipment.Stage(body.Stage)
	if !domainshipment.IsUpdateStage(stage) {
		writeUpdateError(w, ErrInvalidRequest)
		return
	}
	if body.AssignedPartnerID != nil && stage != domainshipment.StageAssignedToPartner {
		writeUpdateError(w, ErrInvalidRequest)
		return
	}

	update, err := h.recorder.Record(r.Context(), code, stage, body.AssignedPartnerID, principal)
	if err != nil {
		writeUpdateError(w, err)
		return
	}

	writeUpdateJSON(w, http.StatusCreated, newUpdateResponse(r.Context(), h.directory, code, update, principal))
}

// handleGroundPartners lists every registered ground partner for any signed-in
// party. The repository supplies the display order; the handler only projects
// the domain values onto the contract and adds no assignment information.
func (h *handler) handleGroundPartners(w http.ResponseWriter, r *http.Request) {
	principal, err := h.principals.Principal(r)
	if err != nil || principal.UserID == "" || principal.Role == "" {
		writeUpdateError(w, ErrSessionExpired)
		return
	}

	partners, err := h.partners.List(r.Context())
	if err != nil {
		writeUpdateError(w, err)
		return
	}

	// A non-nil slice keeps an empty result as [] instead of null.
	list := make([]party, 0, len(partners))
	for _, groundPartner := range partners {
		list = append(list, party{ID: groundPartner.ID, Name: groundPartner.Name})
	}
	writeUpdateJSON(w, http.StatusOK, list)
}

// newUpdateResponse projects the immutable update onto the HTTP contract and
// resolves party names from the injected directory using the stored IDs.
// Name lookup happens only after Record has committed the update: a failed or
// empty lookup must not turn a successful write into an error, so the ID is
// returned as the display-name fallback and the 201 response stays truthful.
func newUpdateResponse(
	ctx context.Context,
	directory PartyDirectory,
	code string,
	update *ledger.Update,
	principal domainshipment.Principal,
) updateResponse {
	responsibleName, err := directory.Name(ctx, update.ResponsiblePartyID)
	if err != nil || responsibleName == "" {
		responsibleName = update.ResponsiblePartyID
	}
	response := updateResponse{
		ID:           update.ID,
		ShipmentCode: code,
		Sequence:     update.Sequence,
		Stage:        update.Stage,
		ResponsibleParty: responsibleParty{
			ID:   update.ResponsiblePartyID,
			Name: responsibleName,
			Role: string(principal.Role),
		},
		RecordedAt:          update.RecordedAt.UTC(),
		PreviousFingerprint: update.PreviousFingerprint,
		Fingerprint:         update.Fingerprint,
	}
	if update.AssignedPartnerID != nil {
		partnerName, err := directory.Name(ctx, *update.AssignedPartnerID)
		if err != nil || partnerName == "" {
			partnerName = *update.AssignedPartnerID
		}
		response.AssignedPartner = &party{ID: *update.AssignedPartnerID, Name: partnerName}
	}
	return response
}
