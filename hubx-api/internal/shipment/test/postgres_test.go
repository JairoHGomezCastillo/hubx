package test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"hubx-api/internal/shipment"
	shipmentmocks "hubx-api/internal/shipment/mocks"
)

const (
	postgresShipmentCode   = "HBX-SEED-0001"
	postgresShipmentID     = "11111111-1111-4111-8111-111111111111"
	postgresCustomerID     = "22222222-2222-4222-8222-222222222222"
	postgresOperatorID     = "33333333-3333-4333-8333-333333333333"
	postgresPartnerID      = "44444444-4444-4444-8444-444444444444"
	postgresOtherPartnerID = "55555555-5555-4555-8555-555555555555"
	postgresPartnerName    = "Transportes Rápido"
	postgresOtherName      = "Logística Express"
)

var errPostgresLookup = errors.New("database lookup failed")

func postgresPtr(value string) *string { return &value }

type capturedQuery struct {
	text string
	args []any
}

func assertPostgresQuery(t *testing.T, got capturedQuery, clause string, wantArgs []any, unwantedLiteral string) {
	t.Helper()
	if !strings.Contains(strings.ToUpper(got.text), clause) {
		t.Errorf("query = %q, want it to contain %q", got.text, clause)
	}
	if !reflect.DeepEqual(got.args, wantArgs) {
		t.Errorf("query args = %#v, want %#v", got.args, wantArgs)
	}
	if unwantedLiteral != "" && strings.Contains(got.text, unwantedLiteral) {
		t.Errorf("query is not parameterized: %q contains %q", got.text, unwantedLiteral)
	}
}

func newPostgresRow(ctrl *gomock.Controller, scan func(...any) error) *shipmentmocks.MockRow {
	row := shipmentmocks.NewMockRow(ctrl)
	row.EXPECT().Scan(gomock.Any()).DoAndReturn(func(dest ...any) error {
		return scan(dest...)
	})
	return row
}

func scanShipmentInto(stored *shipment.Shipment, scanErr error) func(...any) error {
	return func(dest ...any) error {
		if scanErr != nil {
			return scanErr
		}
		if len(dest) != 6 {
			return errors.New("shipment scan has unexpected destinations")
		}
		id, okID := dest[0].(*string)
		code, okCode := dest[1].(*string)
		stage, okStage := dest[2].(*string)
		customerID, okCustomer := dest[3].(*string)
		assignedPartnerID, okPartner := dest[4].(*sql.NullString)
		createdBy, okCreator := dest[5].(*string)
		if !okID || !okCode || !okStage || !okCustomer || !okPartner || !okCreator {
			return errors.New("shipment scan destination has an unexpected type")
		}

		*id = stored.ID
		*code = stored.Code
		*stage = string(stored.Stage)
		*customerID = stored.CustomerID
		*createdBy = stored.CreatedBy
		if stored.AssignedPartnerID != nil {
			assignedPartnerID.Valid = true
			assignedPartnerID.String = *stored.AssignedPartnerID
		}
		return nil
	}
}

func scanBoolInto(value bool, scanErr error) func(...any) error {
	return func(dest ...any) error {
		if scanErr != nil {
			return scanErr
		}
		if len(dest) != 1 {
			return errors.New("boolean scan has unexpected destinations")
		}
		target, ok := dest[0].(*bool)
		if !ok {
			return errors.New("boolean scan destination is not *bool")
		}
		*target = value
		return nil
	}
}

func scanNameInto(name string, scanErr error) func(...any) error {
	return func(dest ...any) error {
		if scanErr != nil {
			return scanErr
		}
		if len(dest) != 1 {
			return errors.New("name scan has unexpected destinations")
		}
		target, ok := dest[0].(*string)
		if !ok {
			return errors.New("name scan destination is not *string")
		}
		*target = name
		return nil
	}
}

func scanPartnerInto(partner shipment.GroundPartner) func(...any) error {
	return func(dest ...any) error {
		if len(dest) != 2 {
			return errors.New("partner scan has unexpected destinations")
		}
		id, okID := dest[0].(*string)
		name, okName := dest[1].(*string)
		if !okID || !okName {
			return errors.New("partner scan destination has an unexpected type")
		}
		*id = partner.ID
		*name = partner.Name
		return nil
	}
}

func TestHU03_FindByCode(t *testing.T) {
	tests := []struct {
		name    string
		stored  *shipment.Shipment
		scanErr error
		want    *shipment.Shipment
		wantErr bool
	}{
		{
			name: "reads a shipment with no assigned partner",
			stored: &shipment.Shipment{
				ID: postgresShipmentID, Code: postgresShipmentCode,
				Stage: shipment.StageCreated, CustomerID: postgresCustomerID,
				CreatedBy: postgresOperatorID,
			},
			want: &shipment.Shipment{
				ID: postgresShipmentID, Code: postgresShipmentCode,
				Stage: shipment.StageCreated, CustomerID: postgresCustomerID,
				CreatedBy: postgresOperatorID,
			},
		},
		{
			name: "reads a shipment with an assigned partner",
			stored: &shipment.Shipment{
				ID: postgresShipmentID, Code: postgresShipmentCode,
				Stage: shipment.StageAssignedToPartner, CustomerID: postgresCustomerID,
				AssignedPartnerID: postgresPtr(postgresPartnerID), CreatedBy: postgresOperatorID,
			},
			want: &shipment.Shipment{
				ID: postgresShipmentID, Code: postgresShipmentCode,
				Stage: shipment.StageAssignedToPartner, CustomerID: postgresCustomerID,
				AssignedPartnerID: postgresPtr(postgresPartnerID), CreatedBy: postgresOperatorID,
			},
		},
		{
			name:    "unknown code returns no shipment and no error",
			scanErr: sql.ErrNoRows,
			want:    nil,
		},
		{
			name:    "scan failure propagates",
			scanErr: errPostgresLookup,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := shipmentmocks.NewMockDB(ctrl)
			row := newPostgresRow(ctrl, scanShipmentInto(tt.stored, tt.scanErr))
			var captured capturedQuery
			db.EXPECT().QueryRowContext(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, query string, args ...any) shipment.Row {
					captured = capturedQuery{text: query, args: append([]any(nil), args...)}
					return row
				},
			)

			repo := shipment.NewPostgresRepository(db)
			got, err := repo.FindByCode(context.Background(), postgresShipmentCode)

			if tt.wantErr {
				if !errors.Is(err, errPostgresLookup) {
					t.Fatalf("FindByCode() error = %v, want %v", err, errPostgresLookup)
				}
				if got != nil {
					t.Errorf("FindByCode() = %+v, want nil on error", got)
				}
			} else {
				if err != nil {
					t.Fatalf("FindByCode() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("FindByCode() = %+v, want %+v", got, tt.want)
				}
			}
			assertPostgresQuery(t, captured, "FROM SHIPMENTS WHERE CODE = $1",
				[]any{postgresShipmentCode}, postgresShipmentCode)
		})
	}
}

func TestHU03_FindByID(t *testing.T) {
	stored := &shipment.Shipment{
		ID: postgresShipmentID, Code: postgresShipmentCode,
		Stage: shipment.StageInFlight, CustomerID: postgresCustomerID,
		CreatedBy: postgresOperatorID,
	}
	tests := []struct {
		name    string
		stored  *shipment.Shipment
		scanErr error
		want    *shipment.Shipment
	}{
		{name: "reads by internal identifier", stored: stored, want: stored},
		{name: "unknown identifier returns no shipment", scanErr: sql.ErrNoRows, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := shipmentmocks.NewMockDB(ctrl)
			row := newPostgresRow(ctrl, scanShipmentInto(tt.stored, tt.scanErr))
			var captured capturedQuery
			db.EXPECT().QueryRowContext(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, query string, args ...any) shipment.Row {
					captured = capturedQuery{text: query, args: append([]any(nil), args...)}
					return row
				},
			)

			repo := shipment.NewPostgresRepository(db)
			got, err := repo.FindByID(context.Background(), postgresShipmentID)

			if err != nil {
				t.Fatalf("FindByID() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindByID() = %+v, want %+v", got, tt.want)
			}
			assertPostgresQuery(t, captured, "FROM SHIPMENTS WHERE ID = $1",
				[]any{postgresShipmentID}, postgresShipmentID)
		})
	}
}

func TestHU03_IsRegisteredPartner(t *testing.T) {
	tests := []struct {
		name    string
		value   bool
		scanErr error
		want    bool
		wantErr bool
	}{
		{name: "registered ground partner", value: true, want: true},
		{name: "customer is not a ground partner", value: false, want: false},
		{name: "lookup failure propagates", scanErr: errPostgresLookup, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := shipmentmocks.NewMockDB(ctrl)
			row := newPostgresRow(ctrl, scanBoolInto(tt.value, tt.scanErr))
			var captured capturedQuery
			db.EXPECT().QueryRowContext(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, query string, args ...any) shipment.Row {
					captured = capturedQuery{text: query, args: append([]any(nil), args...)}
					return row
				},
			)

			repo := shipment.NewPostgresRepository(db)
			got, err := repo.IsRegisteredPartner(context.Background(), postgresPartnerID)

			if tt.wantErr {
				if !errors.Is(err, errPostgresLookup) {
					t.Fatalf("IsRegisteredPartner() error = %v, want %v", err, errPostgresLookup)
				}
			} else if err != nil {
				t.Fatalf("IsRegisteredPartner() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("IsRegisteredPartner() = %t, want %t", got, tt.want)
			}
			assertPostgresQuery(t, captured, "FROM USERS WHERE ID = $1 AND ROLE = $2",
				[]any{postgresPartnerID, string(shipment.RoleGroundPartner)}, postgresPartnerID)
		})
	}
}

func TestHU03_ListGroundPartners(t *testing.T) {
	tests := []struct {
		name     string
		rows     []shipment.GroundPartner
		queryErr error
		rowsErr  error
		want     []shipment.GroundPartner
		wantErr  error
	}{
		{
			name: "lists the partners in repository order",
			rows: []shipment.GroundPartner{
				{ID: postgresPartnerID, Name: postgresPartnerName},
				{ID: postgresOtherPartnerID, Name: postgresOtherName},
			},
			want: []shipment.GroundPartner{
				{ID: postgresPartnerID, Name: postgresPartnerName},
				{ID: postgresOtherPartnerID, Name: postgresOtherName},
			},
		},
		{
			name: "no partners returns a non-nil empty slice",
			want: []shipment.GroundPartner{},
		},
		{
			name:     "query failure propagates",
			queryErr: errPostgresLookup,
			wantErr:  errPostgresLookup,
		},
		{
			name:    "iteration failure propagates",
			rowsErr: errPostgresLookup,
			wantErr: errPostgresLookup,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := shipmentmocks.NewMockDB(ctrl)
			var rows *shipmentmocks.MockRows
			if tt.queryErr == nil {
				rows = shipmentmocks.NewMockRows(ctrl)
				rows.EXPECT().Next().Return(true).Times(len(tt.rows))
				rows.EXPECT().Next().Return(false)
				for _, partner := range tt.rows {
					rows.EXPECT().Scan(gomock.Any()).DoAndReturn(scanPartnerInto(partner))
				}
				rows.EXPECT().Err().Return(tt.rowsErr)
				rows.EXPECT().Close().Return(nil)
			}

			var captured capturedQuery
			db.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, query string, args ...any) (shipment.Rows, error) {
					captured = capturedQuery{text: query, args: append([]any(nil), args...)}
					if tt.queryErr != nil {
						return nil, tt.queryErr
					}
					return rows, nil
				},
			)

			repo := shipment.NewPostgresRepository(db)
			got, err := repo.List(context.Background())

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("List() error = %v, want %v", err, tt.wantErr)
				}
				if got != nil {
					t.Errorf("List() = %+v, want nil on error", got)
				}
			} else {
				if err != nil {
					t.Fatalf("List() unexpected error: %v", err)
				}
				if got == nil {
					t.Fatal("List() = nil, want a non-nil slice")
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("List() = %+v, want %+v", got, tt.want)
				}
			}
			assertPostgresQuery(t, captured, "FROM USERS WHERE ROLE = $1 ORDER BY NAME",
				[]any{string(shipment.RoleGroundPartner)}, "")
		})
	}
}

func TestHU03_Name(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		scanErr error
		want    string
		wantErr bool
	}{
		{name: "reads the user display name", value: postgresPartnerName, want: postgresPartnerName},
		{name: "lookup failure propagates", scanErr: errPostgresLookup, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := shipmentmocks.NewMockDB(ctrl)
			row := newPostgresRow(ctrl, scanNameInto(tt.value, tt.scanErr))
			var captured capturedQuery
			db.EXPECT().QueryRowContext(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, query string, args ...any) shipment.Row {
					captured = capturedQuery{text: query, args: append([]any(nil), args...)}
					return row
				},
			)

			repo := shipment.NewPostgresRepository(db)
			got, err := repo.Name(context.Background(), postgresPartnerID)

			if tt.wantErr {
				if !errors.Is(err, errPostgresLookup) {
					t.Fatalf("Name() error = %v, want %v", err, errPostgresLookup)
				}
				if got != "" {
					t.Errorf("Name() = %q, want empty on error", got)
				}
			} else {
				if err != nil {
					t.Fatalf("Name() unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("Name() = %q, want %q", got, tt.want)
				}
			}
			assertPostgresQuery(t, captured, "SELECT NAME FROM USERS WHERE ID = $1",
				[]any{postgresPartnerID}, postgresPartnerID)
		})
	}
}
