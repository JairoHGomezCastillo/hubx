package shipment

import (
	"context"
	"fmt"

	"hubx-api/internal/ledger"
)

//go:generate go tool mockgen -source=traceability.go -destination=mocks/traceability_mock.go -package=mocks
//go:generate go tool mockgen -source=../ledger/repository.go -destination=mocks/ledger_repository_mock.go -package=mocks

// UpdateView is an update projected for the traceability response.
type UpdateView struct {
	Sequence            int
	Stage               string
	ResponsibleParty    Party
	AssignedPartner     *Party
	RecordedAt          string // RFC3339Nano UTC
	PreviousFingerprint *string
	Fingerprint         string
}

// Traceability is the complete view of a shipment and its update chain.
type Traceability struct {
	ShipmentCode string
	Stage        string
	Updates      []UpdateView
}

// TraceabilityReader reads the traceability of a shipment, applying visibility rules.
type TraceabilityReader interface {
	Read(ctx context.Context, code string, principal Principal) (*Traceability, error)
}

type traceabilityReader struct {
	shipments ShipmentRepository
	ledger    ledger.LedgerRepository
	parties   PartyRepository
}

// NewTraceabilityReader returns a TraceabilityReader with injected repositories.
func NewTraceabilityReader(
	shipments ShipmentRepository,
	ledger ledger.LedgerRepository,
	parties PartyRepository,
) TraceabilityReader {
	return &traceabilityReader{
		shipments: shipments,
		ledger:    ledger,
		parties:   parties,
	}
}

// Read resolves the shipment, checks visibility, reads the update chain and resolves party names.
// Unknown or invisible shipments return ErrShipmentNotFound. Repository errors propagate.
func (r *traceabilityReader) Read(ctx context.Context, code string, principal Principal) (*Traceability, error) {
	current, err := r.shipments.FindByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("find shipment: %w", err)
	}
	if current == nil || !canSee(current, principal) {
		return nil, ErrShipmentNotFound
	}

	updates, err := r.ledger.List(ctx, current.ID)
	if err != nil {
		return nil, fmt.Errorf("list updates: %w", err)
	}

	// Collect distinct party ids to resolve in one query
	partyIDs := make(map[string]struct{})
	for _, update := range updates {
		partyIDs[update.ResponsiblePartyID] = struct{}{}
		if update.AssignedPartnerID != nil {
			partyIDs[*update.AssignedPartnerID] = struct{}{}
		}
	}

	ids := make([]string, 0, len(partyIDs))
	for id := range partyIDs {
		ids = append(ids, id)
	}

	partyMap, err := r.parties.Parties(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("resolve parties: %w", err)
	}

	// Project the updates with resolved names
	views := make([]UpdateView, len(updates))
	for i, update := range updates {
		view := UpdateView{
			Sequence:            update.Sequence,
			Stage:               update.Stage,
			RecordedAt:          update.RecordedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
			PreviousFingerprint: update.PreviousFingerprint,
			Fingerprint:         update.Fingerprint,
		}

		// Resolve responsible party (always present)
		if party, found := partyMap[update.ResponsiblePartyID]; found {
			view.ResponsibleParty = party
		} else {
			// Fallback: use ID as name when lookup fails
			view.ResponsibleParty = Party{
				ID:   update.ResponsiblePartyID,
				Name: update.ResponsiblePartyID,
				Role: "", // Unknown role
			}
		}

		// Resolve assigned partner (only on Assigned to partner updates)
		if update.AssignedPartnerID != nil {
			if party, found := partyMap[*update.AssignedPartnerID]; found {
				view.AssignedPartner = &Party{
					ID:   party.ID,
					Name: party.Name,
					// Role not included in assignedPartner response
				}
			} else {
				view.AssignedPartner = &Party{
					ID:   *update.AssignedPartnerID,
					Name: *update.AssignedPartnerID,
				}
			}
		}

		views[i] = view
	}

	return &Traceability{
		ShipmentCode: current.Code,
		Stage:        string(current.Stage),
		Updates:      views,
	}, nil
}
