package ledger

import (
	"context"
	"time"
)

//go:generate go tool mockgen -source=repository.go -destination=mocks/repository_mock.go -package=mocks

// Assignment represents a shipment assigned to a partner that has not been delivered.
type Assignment struct {
	ShipmentID string
	UpdateID   int64
	AssignedAt time.Time
}

// LedgerRepository is the only gateway to the updates table. Last returns nil when the
// shipment has no update yet. List returns every update of the shipment ordered by
// sequence ascending (oldest first); it returns an empty non-nil slice when none exist.
// Append inserts one update and updates the shipment projection atomically; it never
// modifies or deletes an existing update. PendingAssignments returns shipments assigned
// to the given partner that have no Delivered update, ordered by assignment time then
// update id, with LIMIT and OFFSET for pagination.
type LedgerRepository interface {
	Last(ctx context.Context, shipmentID string) (*Update, error)
	List(ctx context.Context, shipmentID string) ([]Update, error)
	Append(ctx context.Context, in AppendInput) (*Update, error)
	PendingAssignments(ctx context.Context, partnerID string, limit, offset int) ([]Assignment, error)
}
