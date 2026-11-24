package test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"hubx-api/internal/ledger"
	"hubx-api/internal/ledger/mocks"

	"go.uber.org/mock/gomock"
)

// TestHU05_ListReturnsUpdatesInSequenceOrder verifies that List returns all
// updates of a shipment ordered by sequence ascending, with all columns
// including shipment_id.
func TestHU05_ListReturnsUpdatesInSequenceOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := mocks.NewMockDB(ctrl)
	rows := &mockRows{
		data: [][]any{
			{int64(1), "shipment-1", 1, "In flight", "user-1", sql.NullString{}, time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC), sql.NullString{}, "fp1"},
			{int64(2), "shipment-1", 2, "At destination airport", "user-1", sql.NullString{}, time.Date(2026, 11, 25, 11, 0, 0, 0, time.UTC), sql.NullString{Valid: true, String: "fp1"}, "fp2"},
			{int64(3), "shipment-1", 3, "Assigned to partner", "user-1", sql.NullString{Valid: true, String: "partner-1"}, time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC), sql.NullString{Valid: true, String: "fp2"}, "fp3"},
		},
	}

	db.EXPECT().QueryContext(gomock.Any(), gomock.Any(), "shipment-1").Return(rows, nil)

	repo := ledger.NewPostgresRepository(db)
	updates, err := repo.List(context.Background(), "shipment-1")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(updates) != 3 {
		t.Fatalf("got %d updates, want 3", len(updates))
	}

	// Verify order and content
	if updates[0].Sequence != 1 || updates[0].Stage != "In flight" {
		t.Errorf("update[0]: Sequence = %d, Stage = %s; want 1, In flight", updates[0].Sequence, updates[0].Stage)
	}
	if updates[1].Sequence != 2 || updates[1].Stage != "At destination airport" {
		t.Errorf("update[1]: Sequence = %d, Stage = %s; want 2, At destination airport", updates[1].Sequence, updates[1].Stage)
	}
	if updates[2].Sequence != 3 || updates[2].Stage != "Assigned to partner" {
		t.Errorf("update[2]: Sequence = %d, Stage = %s; want 3, Assigned to partner", updates[2].Sequence, updates[2].Stage)
	}

	// Verify shipment_id is populated
	if updates[0].ShipmentID != "shipment-1" {
		t.Errorf("update[0].ShipmentID = %s, want shipment-1", updates[0].ShipmentID)
	}

	// Verify assigned partner is correctly scanned
	if updates[2].AssignedPartnerID == nil || *updates[2].AssignedPartnerID != "partner-1" {
		t.Errorf("update[2].AssignedPartnerID = %v, want partner-1", updates[2].AssignedPartnerID)
	}
}

// TestHU05_ListEmptyChain verifies that List returns an empty slice for a
// shipment with no updates.
func TestHU05_ListEmptyChain(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := mocks.NewMockDB(ctrl)
	rows := &mockRows{data: [][]any{}}

	db.EXPECT().QueryContext(gomock.Any(), gomock.Any(), "shipment-empty").Return(rows, nil)

	repo := ledger.NewPostgresRepository(db)
	updates, err := repo.List(context.Background(), "shipment-empty")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(updates) != 0 {
		t.Errorf("got %d updates for empty chain, want 0", len(updates))
	}
}

// mockRows simulates sql.Rows for testing List.
type mockRows struct {
	data  [][]any
	index int
}

func (r *mockRows) Next() bool {
	return r.index < len(r.data)
}

func (r *mockRows) Scan(dest ...any) error {
	if r.index >= len(r.data) {
		return sql.ErrNoRows
	}
	row := r.data[r.index]
	r.index++
	for i, d := range dest {
		switch v := d.(type) {
		case *int64:
			*v = row[i].(int64)
		case *string:
			*v = row[i].(string)
		case *int:
			*v = row[i].(int)
		case *sql.NullString:
			*v = row[i].(sql.NullString)
		case *time.Time:
			*v = row[i].(time.Time)
		}
	}
	return nil
}

func (r *mockRows) Close() error {
	return nil
}

func (r *mockRows) Err() error {
	return nil
}
