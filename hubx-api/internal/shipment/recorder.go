package shipment

import (
	"context"
	"fmt"
	"time"

	"hubx-api/internal/ledger"
)

// Recorder advances a shipment to the next stage and appends the corresponding
// immutable update to its ledger chain. It is the domain API used by the HTTP
// handler; transport concerns stay outside this package.
type Recorder interface {
	Record(ctx context.Context, code string, requested Stage, partnerID *string, principal Principal) (*ledger.Update, error)
}

type recorder struct {
	shipments ShipmentRepository
	partners  GroundPartnerRepository
	ledger    ledger.LedgerRepository
}

// NewRecorder returns a Recorder with explicitly injected repositories.
func NewRecorder(shipments ShipmentRepository, partners GroundPartnerRepository, ledgerRepo ledger.LedgerRepository) Recorder {
	return &recorder{shipments: shipments, partners: partners, ledger: ledgerRepo}
}

// Record validates the caller, the strict transition and the assignment rules,
// then appends one update chained to the stored predecessor. A rejection never
// writes to the ledger and never changes the shipment projection.
func (r *recorder) Record(
	ctx context.Context,
	code string,
	requested Stage,
	partnerID *string,
	principal Principal,
) (*ledger.Update, error) {
	current, err := r.shipments.FindByCode(ctx, code)
	if err != nil || current == nil || !canSee(current, principal) {
		// FR-012: unknown, invisible and failing lookups are indistinguishable.
		return nil, ErrNotResponsible
	}
	if !IsUpdateStage(requested) {
		return nil, ErrOutOfSequence
	}
	transition, ok := NextTransition(current.Stage)
	if !ok || transition.Next != requested {
		return nil, ErrOutOfSequence
	}
	if principal.Role != transition.Actor {
		return nil, ErrNotResponsible
	}
	if transition.Actor == RoleGroundPartner &&
		(current.AssignedPartnerID == nil || *current.AssignedPartnerID != principal.UserID) {
		return nil, ErrNotResponsible
	}

	var assignedPartnerID *string
	if transition.RequiresPartner {
		if partnerID == nil || *partnerID == "" {
			return nil, ErrPartnerRequired
		}
		registered, err := r.partners.IsRegisteredPartner(ctx, *partnerID)
		if err != nil {
			return nil, fmt.Errorf("check registered ground partner: %w", err)
		}
		if !registered {
			return nil, ErrNotGroundPartner
		}
		assignedPartnerID = partnerID
	}

	input := ledger.AppendInput{
		ShipmentID:         current.ID,
		Sequence:           1,
		Stage:              string(requested),
		ResponsiblePartyID: principal.UserID,
		AssignedPartnerID:  assignedPartnerID,
		RecordedAt:         time.Now().UTC(),
	}
	last, err := r.ledger.Last(ctx, current.ID)
	if err != nil {
		return nil, fmt.Errorf("read last update: %w", err)
	}
	if last != nil {
		input.Sequence = last.Sequence + 1
		fingerprint := last.Fingerprint
		input.PreviousFingerprint = &fingerprint
	}
	if err := ledger.ValidateAppendInput(input); err != nil {
		return nil, err
	}
	if err := ledger.CheckSequence(input.Sequence, last); err != nil {
		return nil, err
	}
	if err := ledger.CheckPredecessor(input, last); err != nil {
		return nil, err
	}
	return r.ledger.Append(ctx, input)
}

// canSee applies the visibility rules of the data model before any stage or
// responsibility check, so a caller cannot probe another party's shipment.
func canSee(current *Shipment, principal Principal) bool {
	switch principal.Role {
	case RoleOperator:
		return true
	case RoleCustomer:
		return current.CustomerID != "" && current.CustomerID == principal.UserID
	case RoleGroundPartner:
		return current.AssignedPartnerID != nil && *current.AssignedPartnerID == principal.UserID
	default:
		return false
	}
}
