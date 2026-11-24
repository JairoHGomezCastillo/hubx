package test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"hubx-api/internal/ledger"
	"hubx-api/internal/ledger/mocks"
)

var errDatabaseDown = errors.New("database down")

func TestServiceAppendChainsToTheStoredUpdate(t *testing.T) {
	recordedAt := time.Date(2026, 11, 24, 10, 15, 30, 123456789, time.UTC)
	draft := ledger.Draft{
		ShipmentID:         shipmentID,
		Stage:              "At destination airport",
		ResponsiblePartyID: responsiblePartyID,
		RecordedAt:         recordedAt,
	}
	stored := &ledger.Update{Sequence: 3, Fingerprint: firstFingerprint}
	wantAppended := &ledger.Update{
		ID:                  7,
		ShipmentID:          shipmentID,
		Sequence:            4,
		Stage:               "At destination airport",
		ResponsiblePartyID:  responsiblePartyID,
		RecordedAt:          recordedAt,
		PreviousFingerprint: ptr(firstFingerprint),
		Fingerprint:         strings.Repeat("c", 64),
	}

	tests := []struct {
		name           string
		draft          ledger.Draft
		last           *ledger.Update
		lastErr        error
		appendErr      error
		wantSequence   int
		wantPrevious   *string
		wantErr        error
		wantLastCall   bool
		wantAppendCall bool
	}{
		{
			name:           "first update has no predecessor",
			draft:          draft,
			wantSequence:   1,
			wantLastCall:   true,
			wantAppendCall: true,
		},
		{
			name:           "later update links to the last fingerprint",
			draft:          draft,
			last:           stored,
			wantSequence:   4,
			wantPrevious:   ptr(firstFingerprint),
			wantLastCall:   true,
			wantAppendCall: true,
		},
		{
			name:         "last lookup failure stops the append",
			draft:        draft,
			lastErr:      errDatabaseDown,
			wantErr:      errDatabaseDown,
			wantLastCall: true,
		},
		{
			name:           "append failure is propagated",
			draft:          draft,
			appendErr:      ledger.ErrConcurrentUpdate,
			wantErr:        ledger.ErrConcurrentUpdate,
			wantSequence:   1,
			wantLastCall:   true,
			wantAppendCall: true,
		},
		{
			name: "invalid draft never reaches the repository",
			draft: ledger.Draft{
				Stage:              "In flight",
				ResponsiblePartyID: responsiblePartyID,
				RecordedAt:         recordedAt,
			},
			wantErr: ledger.ErrInvalidUpdate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockLedgerRepository(ctrl)
			service := ledger.NewService(repo)

			if tt.wantLastCall {
				repo.EXPECT().Last(gomock.Any(), tt.draft.ShipmentID).Return(tt.last, tt.lastErr).Times(1)
			}
			if tt.wantAppendCall {
				repo.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, in ledger.AppendInput) (*ledger.Update, error) {
						if in.Sequence != tt.wantSequence {
							t.Errorf("append sequence = %d, want %d", in.Sequence, tt.wantSequence)
						}
						if (in.PreviousFingerprint == nil) != (tt.wantPrevious == nil) {
							t.Fatalf("previous fingerprint = %v, want %v", in.PreviousFingerprint, tt.wantPrevious)
						}
						if tt.wantPrevious != nil && *in.PreviousFingerprint != *tt.wantPrevious {
							t.Errorf("previous fingerprint = %q, want %q", *in.PreviousFingerprint, *tt.wantPrevious)
						}
						if in.ShipmentID != tt.draft.ShipmentID || in.Stage != tt.draft.Stage {
							t.Errorf("append input = %+v, draft = %+v", in, tt.draft)
						}
						if tt.appendErr != nil {
							return nil, tt.appendErr
						}
						return wantAppended, nil
					},
				).Times(1)
			}

			got, err := service.Append(context.Background(), tt.draft)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				if got != nil {
					t.Errorf("update = %+v, want nil on error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Append() unexpected error: %v", err)
			}
			if got != wantAppended {
				t.Errorf("Append() = %+v, want %+v", got, wantAppended)
			}
		})
	}
}

func TestServiceLastUsesTheRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockLedgerRepository(ctrl)
	last := &ledger.Update{Sequence: 2, Fingerprint: firstFingerprint}
	repo.EXPECT().Last(gomock.Any(), shipmentID).Return(last, nil)

	service := ledger.NewService(repo)
	got, err := service.Last(context.Background(), shipmentID)
	if err != nil {
		t.Fatalf("Last() unexpected error: %v", err)
	}
	if got != last {
		t.Errorf("Last() = %+v, want %+v", got, last)
	}
}
