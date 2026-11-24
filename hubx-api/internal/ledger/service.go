package ledger

import (
	"context"
	"fmt"
)

// Service composes updates into the chain: it reads the stored predecessor, assigns the
// next sequence and delegates the atomic insert to LedgerRepository. It is the domain API
// used by callers that record a shipment update; never write to the updates table directly.
type Service interface {
	Append(ctx context.Context, draft Draft) (*Update, error)
	Last(ctx context.Context, shipmentID string) (*Update, error)
}

type service struct {
	repo LedgerRepository
}

// NewService returns a Service backed by repo.
func NewService(repo LedgerRepository) Service {
	return &service{repo: repo}
}

// Append validates the draft, reads the last stored update and appends the next entry.
// The expected sequence and predecessor are recomputed here from the ledger; the repository
// checks them again inside its transaction so a concurrent writer cannot break the chain.
func (s *service) Append(ctx context.Context, draft Draft) (*Update, error) {
	input := AppendInput{
		ShipmentID:         draft.ShipmentID,
		Sequence:           1,
		Stage:              draft.Stage,
		ResponsiblePartyID: draft.ResponsiblePartyID,
		AssignedPartnerID:  draft.AssignedPartnerID,
		RecordedAt:         draft.RecordedAt,
	}
	if err := ValidateAppendInput(input); err != nil {
		return nil, err
	}

	last, err := s.repo.Last(ctx, draft.ShipmentID)
	if err != nil {
		return nil, fmt.Errorf("read last update: %w", err)
	}
	if last != nil {
		input.Sequence = last.Sequence + 1
		fingerprint := last.Fingerprint
		input.PreviousFingerprint = &fingerprint
	}
	if err := CheckSequence(input.Sequence, last); err != nil {
		return nil, err
	}
	if err := CheckPredecessor(input, last); err != nil {
		return nil, err
	}
	return s.repo.Append(ctx, input)
}

// Last returns the newest update of a shipment, or nil when the chain is empty.
func (s *service) Last(ctx context.Context, shipmentID string) (*Update, error) {
	return s.repo.Last(ctx, shipmentID)
}
