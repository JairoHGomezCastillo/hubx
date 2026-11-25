package evidence

import (
	"context"
	"fmt"
	"time"

	"hubx-api/internal/ledger"
	"hubx-api/internal/shipment"
)

//go:generate go tool mockgen -source=reader.go -destination=mocks/reader_mock.go -package=mocks
//go:generate go tool mockgen -source=../shipment/repository.go -destination=mocks/shipment_repository_mock.go -package=mocks
//go:generate go tool mockgen -source=../ledger/repository.go -destination=mocks/ledger_repository_mock.go -package=mocks

// Reader assembles the evidence of a shipment from current records. It is
// read-only: it never writes to the ledger or moves a shipment to a new stage.
type Reader interface {
	Read(ctx context.Context, code string, principal shipment.Principal) (*Evidence, error)
}

// Option customises a Reader; WithClock makes the generation date deterministic
// in tests without changing the production behaviour.
type Option func(*reader)

// WithClock replaces the source of the document generation instant.
func WithClock(clock func() time.Time) Option {
	return func(r *reader) {
		if clock != nil {
			r.clock = clock
		}
	}
}

type reader struct {
	shipments shipment.ShipmentRepository
	ledger    ledger.LedgerRepository
	parties   shipment.PartyRepository
	clock     func() time.Time
}

// NewReader returns a Reader with explicitly injected repositories. All ledger
// access goes through LedgerRepository (Constitution I) and all SQL in the
// injected repositories is parameterized (Constitution VI).
func NewReader(
	shipments shipment.ShipmentRepository,
	ledgerRepo ledger.LedgerRepository,
	parties shipment.PartyRepository,
	opts ...Option,
) *reader {
	r := &reader{shipments: shipments, ledger: ledgerRepo, parties: parties, clock: time.Now}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Read resolves the shipment, applies the shared visibility rule and assembles
// the document. Unknown and invisible shipments both return ErrShipmentNotFound;
// a caller without the operator role returns ErrForbidden before any lookup, so
// the role refusal cannot reveal whether a shipment exists.
func (r *reader) Read(ctx context.Context, code string, principal shipment.Principal) (*Evidence, error) {
	if principal.Role != shipment.RoleOperator {
		return nil, ErrForbidden
	}

	current, err := r.shipments.FindByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("find shipment: %w", err)
	}
	if current == nil || !shipment.CanSee(current, principal) {
		return nil, ErrShipmentNotFound
	}

	updates, err := r.ledger.List(ctx, current.ID)
	if err != nil {
		return nil, fmt.Errorf("list updates: %w", err)
	}

	ids := make([]string, 0, len(updates)*2)
	for _, update := range updates {
		ids = append(ids, update.ResponsiblePartyID)
		if update.AssignedPartnerID != nil {
			ids = append(ids, *update.AssignedPartnerID)
		}
	}
	parties, err := r.parties.Parties(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("resolve parties: %w", err)
	}

	views := make([]UpdateEvidence, 0, len(updates))
	for _, update := range updates {
		previous := "-"
		if update.PreviousFingerprint != nil {
			previous = *update.PreviousFingerprint
		}
		assignedPartner := ""
		if update.AssignedPartnerID != nil {
			assignedPartner = partyName(parties, *update.AssignedPartnerID)
		}
		input := ledger.AppendInput{
			ShipmentID:          update.ShipmentID,
			Sequence:            update.Sequence,
			Stage:               update.Stage,
			ResponsiblePartyID:  update.ResponsiblePartyID,
			AssignedPartnerID:   update.AssignedPartnerID,
			RecordedAt:          update.RecordedAt.UTC().Truncate(time.Microsecond),
			PreviousFingerprint: update.PreviousFingerprint,
		}
		views = append(views, UpdateEvidence{
			Sequence:            update.Sequence,
			Stage:               update.Stage,
			ResponsibleParty:    partyName(parties, update.ResponsiblePartyID),
			AssignedPartner:     assignedPartner,
			RecordedAt:          update.RecordedAt.UTC(),
			Fingerprint:         update.Fingerprint,
			PreviousFingerprint: previous,
			CanonicalInput:      ledger.CanonicalInput(input),
		})
	}

	return &Evidence{
		Shipment: ShipmentData{
			Code:         current.Code,
			Origin:       current.Origin,
			Destination:  current.Destination,
			Flight:       current.Flight,
			WeightKg:     current.WeightKg,
			CustomerName: current.CustomerName,
			Stage:        string(current.Stage),
		},
		Updates:      views,
		GeneratedAt:  r.clock().UTC(),
		Instructions: Instructions,
	}, nil
}

func partyName(parties map[string]shipment.Party, id string) string {
	if party, found := parties[id]; found && party.Name != "" {
		return party.Name
	}
	return id
}
