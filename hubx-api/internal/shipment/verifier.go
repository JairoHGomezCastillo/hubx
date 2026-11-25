package shipment

import (
	"context"
	"fmt"

	"hubx-api/internal/ledger"
)

//go:generate go tool mockgen -source=verifier.go -destination=mocks/verifier_mock.go -package=mocks

// Verification is the result of verifying a shipment's update chain.
type Verification struct {
	ShipmentCode       string
	Intact             bool
	UpdatesChecked     int
	FirstAlteredUpdate int // 1-based position, 0 if intact
}

// Verifier verifies the integrity of a shipment's ledger chain, applying the
// same visibility rules as the recorder (operator: any; customer: own;
// ground partner: assigned).
type Verifier interface {
	Verify(ctx context.Context, code string, principal Principal) (*Verification, error)
}

type verifier struct {
	shipments ShipmentRepository
	ledger    ledger.LedgerRepository
}

// NewVerifier returns a Verifier with explicitly injected repositories.
func NewVerifier(shipments ShipmentRepository, ledgerRepo ledger.LedgerRepository) Verifier {
	return &verifier{shipments: shipments, ledger: ledgerRepo}
}

// Verify finds the shipment by code, checks visibility, reads all its updates
// and returns the verification result. Unknown and invisible shipments both
// return ErrShipmentNotFound with an identical response.
func (v *verifier) Verify(ctx context.Context, code string, principal Principal) (*Verification, error) {
	ship, err := v.shipments.FindByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("find shipment by code: %w", err)
	}
	if ship == nil || !canSee(ship, principal) {
		return nil, ErrShipmentNotFound
	}

	updates, err := v.ledger.List(ctx, ship.ID)
	if err != nil {
		return nil, fmt.Errorf("list updates: %w", err)
	}

	result := ledger.Verify(updates)

	return &Verification{
		ShipmentCode:       ship.Code,
		Intact:             result.Intact,
		UpdatesChecked:     result.UpdatesChecked,
		FirstAlteredUpdate: result.FirstAlteredUpdate,
	}, nil
}
