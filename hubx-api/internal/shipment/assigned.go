package shipment

import (
	"context"
	"errors"
	"time"

	"hubx-api/internal/ledger"
)

const (
	// PageSize is the number of shipments returned per page.
	PageSize = 20
	// MaxOffset is the highest allowed offset for pagination.
	MaxOffset = 100000
)

var (
	// ErrNoPermission is returned when a non-ground-partner tries to list assignments.
	ErrNoPermission = errors.New("forbidden: only ground partners can view assigned shipments")

	// ErrInvalidOffset is returned when offset is out of bounds.
	ErrInvalidOffset = errors.New("invalid offset")
)

// AssignedShipment represents one shipment in the partner's assigned list.
type AssignedShipment struct {
	Code        string
	Stage       Stage
	Origin      string
	Destination string
	Flight      string
	WeightKg    string
	AssignedAt  time.Time
}

// AssignedPage is the paginated result of listing assigned shipments.
type AssignedPage struct {
	Shipments  []AssignedShipment
	NextOffset *int
	Message    *string
}

// AssignedLister lists shipments assigned to a partner that are not yet delivered.
type AssignedLister interface {
	List(ctx context.Context, principal Principal, offset int) (AssignedPage, error)
}

type assignedLister struct {
	ledger    ledger.LedgerRepository
	shipments ShipmentRepository
}

// NewAssignedLister returns an AssignedLister that reads from the ledger and shipments.
func NewAssignedLister(ledger ledger.LedgerRepository, shipments ShipmentRepository) AssignedLister {
	return &assignedLister{ledger: ledger, shipments: shipments}
}

// List returns the partner's pending assignments, oldest assignment first, 20 at a time.
// Role is checked first, then offset bounds. It requests PageSize+1 rows to compute nextOffset.
func (l *assignedLister) List(ctx context.Context, principal Principal, offset int) (AssignedPage, error) {
	// Check role before offset
	if principal.Role != RoleGroundPartner {
		return AssignedPage{}, ErrNoPermission
	}

	// Check offset bounds
	if offset < 0 || offset > MaxOffset {
		return AssignedPage{}, ErrInvalidOffset
	}

	// Request PageSize+1 to detect if there is a next page
	assignments, err := l.ledger.PendingAssignments(ctx, principal.UserID, PageSize+1, offset)
	if err != nil {
		return AssignedPage{}, err
	}

	// Extract shipment IDs in the order returned by the ledger
	hasMore := len(assignments) > PageSize
	if hasMore {
		assignments = assignments[:PageSize]
	}

	if len(assignments) == 0 {
		msg := "You have no assigned shipments"
		return AssignedPage{Shipments: []AssignedShipment{}, Message: &msg}, nil
	}

	// Build ID list in ledger order
	ids := make([]string, len(assignments))
	for i, a := range assignments {
		ids[i] = a.ShipmentID
	}

	// Fetch shipment details
	shipments, err := l.shipments.FindByIDs(ctx, ids)
	if err != nil {
		return AssignedPage{}, err
	}

	// Build a map for quick lookup
	shipmentMap := make(map[string]Shipment, len(shipments))
	for _, s := range shipments {
		shipmentMap[s.ID] = s
	}

	// Build result in ledger order, preserving assignedAt from the assignment update
	result := make([]AssignedShipment, 0, len(assignments))
	for _, a := range assignments {
		s, ok := shipmentMap[a.ShipmentID]
		if !ok {
			// Shipment missing from join (should not happen); skip
			continue
		}
		result = append(result, AssignedShipment{
			Code:        s.Code,
			Stage:       s.Stage,
			Origin:      s.Origin,
			Destination: s.Destination,
			Flight:      s.Flight,
			WeightKg:    s.WeightKg,
			AssignedAt:  a.AssignedAt,
		})
	}

	var nextOffset *int
	if hasMore {
		next := offset + PageSize
		nextOffset = &next
	}

	return AssignedPage{Shipments: result, NextOffset: nextOffset}, nil
}
