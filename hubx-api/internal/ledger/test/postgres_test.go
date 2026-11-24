package test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/mock/gomock"

	"hubx-api/internal/ledger"
	"hubx-api/internal/ledger/mocks"
)

const insertedUpdateID = int64(7)

var _ ledger.LedgerRepository = ledger.NewPostgresRepository(nil)

type sqlMatcher struct {
	text string
	seen *[]string
}

func (m sqlMatcher) Matches(x any) bool {
	query, ok := x.(string)
	if !ok {
		return false
	}
	if m.seen != nil {
		*m.seen = append(*m.seen, query)
	}
	return strings.Contains(strings.ToUpper(query), m.text)
}

func (m sqlMatcher) String() string { return "SQL containing " + m.text }

type argsMatcher struct {
	seen *[]any
}

func (m argsMatcher) Matches(x any) bool {
	args, ok := x.([]any)
	if !ok {
		return false
	}
	*m.seen = args
	return true
}

func (m argsMatcher) String() string { return "capture variadic arguments" }

type captureArgMatcher struct {
	seen *[]any
}

func (m captureArgMatcher) Matches(x any) bool {
	*m.seen = append(*m.seen, x)
	return true
}

func (m captureArgMatcher) String() string { return "capture argument" }

func scanLockedShipment(want string) func(...any) error {
	return func(dest ...any) error {
		if len(dest) != 1 {
			return errors.New("shipment lock scan has unexpected destinations")
		}
		target, ok := dest[0].(*string)
		if !ok {
			return errors.New("shipment lock scan destination is not *string")
		}
		*target = want
		return nil
	}
}

func scanStoredUpdate(last *ledger.Update) func(...any) error {
	return func(dest ...any) error {
		if last == nil {
			return sql.ErrNoRows
		}
		if len(dest) != 8 {
			return errors.New("last update scan has unexpected destinations")
		}
		id, ok := dest[0].(*int64)
		if !ok {
			return errors.New("last update id destination is not *int64")
		}
		sequence, ok := dest[1].(*int)
		if !ok {
			return errors.New("last update sequence destination is not *int")
		}
		stage, ok := dest[2].(*string)
		if !ok {
			return errors.New("last update stage destination is not *string")
		}
		responsiblePartyID, ok := dest[3].(*string)
		if !ok {
			return errors.New("last update responsible party destination is not *string")
		}
		assignedPartnerID, ok := dest[4].(*sql.NullString)
		if !ok {
			return errors.New("last update assigned partner destination is not *sql.NullString")
		}
		recordedAt, ok := dest[5].(*time.Time)
		if !ok {
			return errors.New("last update date destination is not *time.Time")
		}
		previousFingerprint, ok := dest[6].(*sql.NullString)
		if !ok {
			return errors.New("last update predecessor destination is not *sql.NullString")
		}
		fingerprint, ok := dest[7].(*string)
		if !ok {
			return errors.New("last update fingerprint destination is not *string")
		}

		*id = last.ID
		*sequence = last.Sequence
		*stage = last.Stage
		*responsiblePartyID = last.ResponsiblePartyID
		*recordedAt = last.RecordedAt
		*fingerprint = last.Fingerprint
		if last.AssignedPartnerID != nil {
			assignedPartnerID.Valid = true
			assignedPartnerID.String = *last.AssignedPartnerID
		}
		if last.PreviousFingerprint != nil {
			previousFingerprint.Valid = true
			previousFingerprint.String = *last.PreviousFingerprint
		}
		return nil
	}
}

func scanInsertedUpdate(id int64) func(...any) error {
	return func(dest ...any) error {
		if len(dest) != 1 {
			return errors.New("insert scan has unexpected destinations")
		}
		target, ok := dest[0].(*int64)
		if !ok {
			return errors.New("insert scan destination is not *int64")
		}
		*target = id
		return nil
	}
}

func newLedgerRow(ctrl *gomock.Controller, scan func(...any) error) *mocks.MockRow {
	row := mocks.NewMockRow(ctrl)
	row.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
		return scan(dest...)
	})
	return row
}

func TestPostgresAppend(t *testing.T) {
	recordedAt := time.Date(2026, 11, 24, 10, 15, 30, 123456789, time.UTC)
	partner := assignedPartnerID
	last := &ledger.Update{
		ID:                 6,
		ShipmentID:         shipmentID,
		Sequence:           3,
		Stage:              "Assigned to partner",
		ResponsiblePartyID: responsiblePartyID,
		AssignedPartnerID:  &partner,
		RecordedAt:         recordedAt.Add(-time.Hour),
		Fingerprint:        firstFingerprint,
	}

	tests := []struct {
		name         string
		in           ledger.AppendInput
		last         *ledger.Update
		insertErr    error
		wantInsert   bool
		wantCommit   bool
		wantRollback bool
		wantErr      error
	}{
		{
			name: "chained update inserts and updates the projection",
			in: ledger.AppendInput{
				ShipmentID:          shipmentID,
				Sequence:            4,
				Stage:               "Out for delivery",
				ResponsiblePartyID:  responsiblePartyID,
				RecordedAt:          recordedAt,
				PreviousFingerprint: ptr(firstFingerprint),
			},
			last:       last,
			wantInsert: true,
			wantCommit: true,
		},
		{
			name: "stale sequence is rejected before insert",
			in: ledger.AppendInput{
				ShipmentID:          shipmentID,
				Sequence:            2,
				Stage:               "Out for delivery",
				ResponsiblePartyID:  responsiblePartyID,
				RecordedAt:          recordedAt,
				PreviousFingerprint: ptr(firstFingerprint),
			},
			last:         last,
			wantRollback: true,
			wantErr:      ledger.ErrConcurrentUpdate,
		},
		{
			name: "unique violation maps to ErrConcurrentUpdate",
			in: ledger.AppendInput{
				ShipmentID:          shipmentID,
				Sequence:            4,
				Stage:               "Out for delivery",
				ResponsiblePartyID:  responsiblePartyID,
				RecordedAt:          recordedAt,
				PreviousFingerprint: ptr(firstFingerprint),
			},
			last:         last,
			insertErr:    &pgconn.PgError{Code: "23505"},
			wantInsert:   true,
			wantRollback: true,
			wantErr:      ledger.ErrConcurrentUpdate,
		},
		{
			name: "serialization failure maps to ErrConcurrentUpdate",
			in: ledger.AppendInput{
				ShipmentID:          shipmentID,
				Sequence:            4,
				Stage:               "Out for delivery",
				ResponsiblePartyID:  responsiblePartyID,
				RecordedAt:          recordedAt,
				PreviousFingerprint: ptr(firstFingerprint),
			},
			last:         last,
			insertErr:    &pgconn.PgError{Code: "40001"},
			wantInsert:   true,
			wantRollback: true,
			wantErr:      ledger.ErrConcurrentUpdate,
		},
		{
			name: "deadlock detection maps to ErrConcurrentUpdate",
			in: ledger.AppendInput{
				ShipmentID:          shipmentID,
				Sequence:            4,
				Stage:               "Out for delivery",
				ResponsiblePartyID:  responsiblePartyID,
				RecordedAt:          recordedAt,
				PreviousFingerprint: ptr(firstFingerprint),
			},
			last:         last,
			insertErr:    &pgconn.PgError{Code: "40P01"},
			wantInsert:   true,
			wantRollback: true,
			wantErr:      ledger.ErrConcurrentUpdate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := mocks.NewMockDB(ctrl)
			tx := mocks.NewMockTx(ctrl)
			var queries []string
			var insertArgs []any

			db.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(tx, nil)

			lockRow := newLedgerRow(ctrl, scanLockedShipment(shipmentID))
			tx.EXPECT().QueryRowContext(gomock.Any(), sqlMatcher{"FOR UPDATE", &queries}, shipmentID).Return(lockRow)

			lastRow := newLedgerRow(ctrl, scanStoredUpdate(tt.last))
			tx.EXPECT().QueryRowContext(gomock.Any(), sqlMatcher{"FROM UPDATES", &queries}, shipmentID).Return(lastRow)

			if tt.wantInsert {
				insertRow := newLedgerRow(ctrl, func(dest ...any) error {
					if tt.insertErr != nil {
						return tt.insertErr
					}
					return scanInsertedUpdate(insertedUpdateID)(dest...)
				})
				tx.EXPECT().QueryRowContext(
					gomock.Any(), sqlMatcher{"INSERT INTO UPDATES", &queries},
					captureArgMatcher{&insertArgs}, captureArgMatcher{&insertArgs},
					captureArgMatcher{&insertArgs}, captureArgMatcher{&insertArgs},
					captureArgMatcher{&insertArgs}, captureArgMatcher{&insertArgs},
					captureArgMatcher{&insertArgs}, captureArgMatcher{&insertArgs},
				).Return(insertRow)
				if tt.insertErr == nil {
					result := mocks.NewMockResult(ctrl)
					tx.EXPECT().ExecContext(
						gomock.Any(), sqlMatcher{"UPDATE SHIPMENTS", &queries},
						gomock.Any(), gomock.Any(), gomock.Any(),
					).Return(result, nil)
				}
			}

			if tt.wantCommit {
				tx.EXPECT().Commit().Return(nil)
			}
			if tt.wantRollback {
				tx.EXPECT().Rollback().Return(nil)
			}

			repo := ledger.NewPostgresRepository(db)
			got, err := repo.Append(context.Background(), tt.in)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Append() error = %v, want %v", err, tt.wantErr)
				}
				if got != nil {
					t.Errorf("Append() update = %+v, want nil on error", got)
				}
			} else {
				if err != nil {
					t.Fatalf("Append() unexpected error: %v", err)
				}
				want := &ledger.Update{
					ID:                  insertedUpdateID,
					ShipmentID:          tt.in.ShipmentID,
					Sequence:            tt.in.Sequence,
					Stage:               tt.in.Stage,
					ResponsiblePartyID:  tt.in.ResponsiblePartyID,
					AssignedPartnerID:   tt.in.AssignedPartnerID,
					RecordedAt:          tt.in.RecordedAt.UTC(),
					PreviousFingerprint: tt.in.PreviousFingerprint,
					Fingerprint:         ledger.Fingerprint(tt.in),
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("Append() = %+v, want %+v", got, want)
				}
			}

			assertAppendSQL(t, queries, insertArgs, tt.in, tt.wantInsert, tt.wantCommit)
		})
	}
}

func TestPostgresLast(t *testing.T) {
	recordedAt := time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	stored := &ledger.Update{
		ID:                 3,
		Sequence:           2,
		Stage:              "In flight",
		ResponsiblePartyID: responsiblePartyID,
		RecordedAt:         recordedAt,
		Fingerprint:        firstFingerprint,
	}

	tests := []struct {
		name string
		last *ledger.Update
	}{
		{name: "newest update", last: stored},
		{name: "empty chain", last: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := mocks.NewMockDB(ctrl)
			row := newLedgerRow(ctrl, scanStoredUpdate(tt.last))
			var queries []string
			db.EXPECT().QueryRowContext(gomock.Any(), sqlMatcher{"FROM UPDATES", &queries}, shipmentID).Return(row)

			repo := ledger.NewPostgresRepository(db)
			got, err := repo.Last(context.Background(), shipmentID)
			if err != nil {
				t.Fatalf("Last() unexpected error: %v", err)
			}
			for _, query := range queries {
				if strings.Contains(query, shipmentID) {
					t.Errorf("query is not parameterized: %q", query)
				}
			}
			if !reflect.DeepEqual(got, tt.last) {
				t.Errorf("Last() = %+v, want %+v", got, tt.last)
			}
		})
	}
}

func assertAppendSQL(t *testing.T, queries []string, insertArgs []any, in ledger.AppendInput, wantInsert, wantProjection bool) {
	t.Helper()
	joined := strings.ToUpper(strings.Join(queries, "\n"))
	if !strings.Contains(joined, "FOR UPDATE") {
		t.Errorf("missing FOR UPDATE lock in queries: %q", queries)
	}
	if strings.Contains(joined, "UPDATE UPDATES") || strings.Contains(joined, "DELETE FROM UPDATES") {
		t.Errorf("updates table must be insert-only, queries: %q", queries)
	}
	for _, query := range queries {
		if strings.Contains(query, shipmentID) {
			t.Errorf("query is not parameterized: %q", query)
		}
	}
	if !wantInsert {
		if strings.Contains(joined, "INSERT INTO UPDATES") {
			t.Errorf("rejected append must not insert: %q", queries)
		}
		return
	}
	if !strings.Contains(joined, "INSERT INTO UPDATES") {
		t.Errorf("missing insert into updates: %q", queries)
	}
	if wantProjection && !strings.Contains(joined, "COALESCE($2, ASSIGNED_PARTNER_ID)") {
		t.Errorf("projection must keep an assigned partner: %q", queries)
	}
	if len(insertArgs) != 8 {
		t.Fatalf("insert has %d arguments, want 8", len(insertArgs))
	}
	if insertArgs[0] != in.ShipmentID {
		t.Errorf("insert shipment = %v, want %q", insertArgs[0], in.ShipmentID)
	}
	if insertArgs[1] != in.Sequence {
		t.Errorf("insert sequence = %v, want %d", insertArgs[1], in.Sequence)
	}
	if insertArgs[7] != ledger.Fingerprint(in) {
		t.Errorf("insert fingerprint = %v, want %q", insertArgs[7], ledger.Fingerprint(in))
	}
}

func TestHU04_List(t *testing.T) {
	recordedAt1 := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)
	recordedAt2 := time.Date(2026, 11, 25, 11, 0, 0, 0, time.UTC)
	recordedAt3 := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	partner := assignedPartnerID

	update1 := ledger.Update{
		ID:                  1,
		ShipmentID:          shipmentID,
		Sequence:            1,
		Stage:               "In flight",
		ResponsiblePartyID:  responsiblePartyID,
		RecordedAt:          recordedAt1,
		Fingerprint:         firstFingerprint,
		PreviousFingerprint: nil,
	}
	update2 := ledger.Update{
		ID:                  2,
		ShipmentID:          shipmentID,
		Sequence:            2,
		Stage:               "At destination airport",
		ResponsiblePartyID:  responsiblePartyID,
		RecordedAt:          recordedAt2,
		Fingerprint:         "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		PreviousFingerprint: ptr(firstFingerprint),
	}
	update3 := ledger.Update{
		ID:                  3,
		ShipmentID:          shipmentID,
		Sequence:            3,
		Stage:               "Assigned to partner",
		ResponsiblePartyID:  responsiblePartyID,
		AssignedPartnerID:   &partner,
		RecordedAt:          recordedAt3,
		Fingerprint:         "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		PreviousFingerprint: ptr("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
	}

	tests := []struct {
		name       string
		setupRows  func(ctrl *gomock.Controller) *mocks.MockRows
		setupError error
		want       []ledger.Update
		wantErr    bool
	}{
		{
			name: "returns three updates ordered by sequence ascending",
			setupRows: func(ctrl *gomock.Controller) *mocks.MockRows {
				rows := mocks.NewMockRows(ctrl)
				callNum := 0
				rows.EXPECT().Next().DoAndReturn(func() bool {
					callNum++
					return callNum <= 3
				}).Times(4)
				scanNum := 0
				rows.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
					scanNum++
					var update *ledger.Update
					switch scanNum {
					case 1:
						update = &update1
					case 2:
						update = &update2
					case 3:
						update = &update3
					}
					if len(dest) != 9 {
						return errors.New("List scan has unexpected destinations")
					}
					*dest[0].(*int64) = update.ID
					*dest[1].(*string) = update.ShipmentID
					*dest[2].(*int) = update.Sequence
					*dest[3].(*string) = update.Stage
					*dest[4].(*string) = update.ResponsiblePartyID
					if update.AssignedPartnerID != nil {
						dest[5].(*sql.NullString).Valid = true
						dest[5].(*sql.NullString).String = *update.AssignedPartnerID
					}
					*dest[6].(*time.Time) = update.RecordedAt
					if update.PreviousFingerprint != nil {
						dest[7].(*sql.NullString).Valid = true
						dest[7].(*sql.NullString).String = *update.PreviousFingerprint
					}
					*dest[8].(*string) = update.Fingerprint
					return nil
				}).Times(3)
				rows.EXPECT().Err().Return(nil)
				rows.EXPECT().Close().Return(nil)
				return rows
			},
			want: []ledger.Update{update1, update2, update3},
		},
		{
			name: "returns empty non-nil slice when no updates exist",
			setupRows: func(ctrl *gomock.Controller) *mocks.MockRows {
				rows := mocks.NewMockRows(ctrl)
				rows.EXPECT().Next().Return(false)
				rows.EXPECT().Err().Return(nil)
				rows.EXPECT().Close().Return(nil)
				return rows
			},
			want: []ledger.Update{},
		},
		{
			name: "scan error propagates",
			setupRows: func(ctrl *gomock.Controller) *mocks.MockRows {
				rows := mocks.NewMockRows(ctrl)
				rows.EXPECT().Next().Return(true)
				rows.EXPECT().Scan(gomock.Any()).Return(errors.New("scan failed"))
				rows.EXPECT().Close().Return(nil)
				return rows
			},
			wantErr: true,
		},
		{
			name: "rows Err() propagates",
			setupRows: func(ctrl *gomock.Controller) *mocks.MockRows {
				rows := mocks.NewMockRows(ctrl)
				rows.EXPECT().Next().Return(false)
				rows.EXPECT().Err().Return(errors.New("iteration failed"))
				rows.EXPECT().Close().Return(nil)
				return rows
			},
			wantErr: true,
		},
		{
			name:       "query error propagates",
			setupError: errors.New("query failed"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := mocks.NewMockDB(ctrl)
			var queries []string

			if tt.setupError != nil {
				db.EXPECT().QueryContext(gomock.Any(), sqlMatcher{"FROM UPDATES", &queries}, shipmentID).Return(nil, tt.setupError)
			} else {
				rows := tt.setupRows(ctrl)
				db.EXPECT().QueryContext(gomock.Any(), sqlMatcher{"FROM UPDATES", &queries}, shipmentID).Return(rows, nil)
			}

			repo := ledger.NewPostgresRepository(db)
			got, err := repo.List(context.Background(), shipmentID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("List() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("List() unexpected error: %v", err)
			}

			// Verify query is parameterized and ordered
			if len(queries) > 0 {
				query := strings.ToUpper(queries[0])
				if strings.Contains(queries[0], shipmentID) {
					t.Errorf("query is not parameterized: %q", queries[0])
				}
				if !strings.Contains(query, "ORDER BY SEQUENCE ASC") {
					t.Errorf("query must order by sequence ascending: %q", queries[0])
				}
			}

			if got == nil {
				t.Fatal("List() returned nil, want non-nil slice")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("List() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
