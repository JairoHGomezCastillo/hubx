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

// TestHU05_AppendTruncatesToMicroseconds verifies that Append truncates the
// instant to microseconds before computing the fingerprint, so the stored
// value and the recomputed fingerprint match even when PostgreSQL drops
// nanoseconds (HU-03 contract rule 1 clarification).
func TestHU05_AppendTruncatesToMicroseconds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := mocks.NewMockDB(ctrl)
	tx := mocks.NewMockTx(ctrl)
	lockRow := mocks.NewMockRow(ctrl)
	lastRow := mocks.NewMockRow(ctrl)
	insertRow := mocks.NewMockRow(ctrl)

	// A nanosecond-precision instant
	inputTime := time.Date(2026, 11, 25, 10, 15, 30, 123456789, time.UTC)
	// The microsecond-truncated instant that should be stored and fingerprinted
	truncatedTime := time.Date(2026, 11, 25, 10, 15, 30, 123456000, time.UTC)

	db.EXPECT().BeginTx(gomock.Any(), nil).Return(tx, nil)
	tx.EXPECT().QueryRowContext(gomock.Any(), gomock.Any(), "shipment-1").Return(lockRow)
	lockRow.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
		if id, ok := dest[0].(*string); ok {
			*id = "shipment-1"
		}
		return nil
	})

	tx.EXPECT().QueryRowContext(gomock.Any(), gomock.Any(), "shipment-1").Return(lastRow)
	lastRow.EXPECT().Scan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(sql.ErrNoRows) // empty chain

	// Verify that the stored recordedAt is the truncated instant
	tx.EXPECT().QueryRowContext(gomock.Any(), gomock.Any(),
		"shipment-1", 1, "In flight", "user-1", nil,
		truncatedTime, // must be microsecond-truncated
		nil, gomock.Any()).Return(insertRow)
	insertRow.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
		if id, ok := dest[0].(*int64); ok {
			*id = 1
		}
		return nil
	})

	tx.EXPECT().ExecContext(gomock.Any(), gomock.Any(), "In flight", nil, "shipment-1").Return(nil, nil)
	tx.EXPECT().Commit().Return(nil)

	repo := ledger.NewPostgresRepository(db)
	input := ledger.AppendInput{
		ShipmentID:          "shipment-1",
		Sequence:            1,
		Stage:               "In flight",
		ResponsiblePartyID:  "user-1",
		AssignedPartnerID:   nil,
		RecordedAt:          inputTime, // nanosecond precision
		PreviousFingerprint: nil,
	}

	update, err := repo.Append(context.Background(), input)
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Verify the returned instant is truncated to microseconds
	if !update.RecordedAt.Equal(truncatedTime) {
		t.Errorf("RecordedAt = %v, want %v", update.RecordedAt, truncatedTime)
	}

	// Verify the fingerprint is computed over the truncated instant
	expectedInput := input
	expectedInput.RecordedAt = truncatedTime
	expectedFingerprint := ledger.Fingerprint(expectedInput)
	if update.Fingerprint != expectedFingerprint {
		t.Errorf("Fingerprint = %s, want %s", update.Fingerprint, expectedFingerprint)
	}

	// Verify that Verify over the returned update with the truncated instant reports intact
	result := ledger.Verify([]ledger.Update{*update})
	if !result.Intact {
		t.Errorf("Verify after Append with microsecond truncation reported altered, want intact")
	}
}
