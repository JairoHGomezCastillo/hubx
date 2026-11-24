package ledger

import "context"

//go:generate go tool mockgen -source=repository.go -destination=mocks/repository_mock.go -package=mocks

// LedgerRepository is the only gateway to the updates table. Last returns nil when the
// shipment has no update yet. List returns every update of the shipment ordered by
// sequence ascending (oldest first); it returns an empty non-nil slice when none exist.
// Append inserts one update and updates the shipment projection atomically; it never
// modifies or deletes an existing update.
type LedgerRepository interface {
	Last(ctx context.Context, shipmentID string) (*Update, error)
	List(ctx context.Context, shipmentID string) ([]Update, error)
	Append(ctx context.Context, in AppendInput) (*Update, error)
}
