package test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"hubx-api/internal/ledger"
	ledgermocks "hubx-api/internal/ledger/mocks"
	"hubx-api/internal/shipment"
	"hubx-api/internal/shipment/mocks"
)

func TestHU07_AssignedLister_RoleCheck(t *testing.T) {
	tests := []struct {
		name      string
		principal shipment.Principal
		wantErr   error
	}{
		{
			name:      "ground partner can list",
			principal: shipment.Principal{UserID: "partner-1", Role: shipment.RoleGroundPartner},
			wantErr:   nil,
		},
		{
			name:      "operator gets ErrNoPermission",
			principal: shipment.Principal{UserID: "operator-1", Role: shipment.RoleOperator},
			wantErr:   shipment.ErrNoPermission,
		},
		{
			name:      "customer gets ErrNoPermission",
			principal: shipment.Principal{UserID: "customer-1", Role: shipment.RoleCustomer},
			wantErr:   shipment.ErrNoPermission,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)
			shipmentRepo := mocks.NewMockShipmentRepository(ctrl)

			if tt.wantErr == nil {
				// Ground partner: expect ledger call
				ledgerRepo.EXPECT().PendingAssignments(gomock.Any(), tt.principal.UserID, 21, 0).
					Return([]ledger.Assignment{}, nil)
			}

			lister := shipment.NewAssignedLister(ledgerRepo, shipmentRepo)
			page, err := lister.List(context.Background(), tt.principal, 0)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("List() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("List() unexpected error: %v", err)
			}
			if page.Shipments == nil {
				t.Error("List() shipments should be non-nil slice")
			}
		})
	}
}

func TestHU07_AssignedLister_OffsetBounds(t *testing.T) {
	principal := shipment.Principal{UserID: "partner-1", Role: shipment.RoleGroundPartner}

	tests := []struct {
		name    string
		offset  int
		wantErr error
	}{
		{name: "offset 0 is valid", offset: 0, wantErr: nil},
		{name: "offset 100000 is valid", offset: 100000, wantErr: nil},
		{name: "offset -1 is invalid", offset: -1, wantErr: shipment.ErrInvalidOffset},
		{name: "offset 100001 is invalid", offset: 100001, wantErr: shipment.ErrInvalidOffset},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)
			shipmentRepo := mocks.NewMockShipmentRepository(ctrl)

			if tt.wantErr == nil {
				ledgerRepo.EXPECT().PendingAssignments(gomock.Any(), principal.UserID, 21, tt.offset).
					Return([]ledger.Assignment{}, nil)
			}

			lister := shipment.NewAssignedLister(ledgerRepo, shipmentRepo)
			_, err := lister.List(context.Background(), principal, tt.offset)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("List() error = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("List() unexpected error: %v", err)
			}
		})
	}
}

func TestHU07_AssignedLister_EmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)
	shipmentRepo := mocks.NewMockShipmentRepository(ctrl)

	principal := shipment.Principal{UserID: "partner-1", Role: shipment.RoleGroundPartner}

	ledgerRepo.EXPECT().PendingAssignments(gomock.Any(), principal.UserID, 21, 0).
		Return([]ledger.Assignment{}, nil)

	lister := shipment.NewAssignedLister(ledgerRepo, shipmentRepo)
	page, err := lister.List(context.Background(), principal, 0)

	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}
	if page.Shipments == nil || len(page.Shipments) != 0 {
		t.Errorf("List() shipments = %v, want empty non-nil slice", page.Shipments)
	}
	if page.NextOffset != nil {
		t.Errorf("List() nextOffset = %v, want nil", page.NextOffset)
	}
	if page.Message == nil || *page.Message != "You have no assigned shipments" {
		t.Errorf("List() message = %v, want 'You have no assigned shipments'", page.Message)
	}
}

func TestHU07_AssignedLister_PageSizePlusOne(t *testing.T) {
	ctrl := gomock.NewController(t)
	ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)
	shipmentRepo := mocks.NewMockShipmentRepository(ctrl)

	principal := shipment.Principal{UserID: "partner-1", Role: shipment.RoleGroundPartner}

	// Ledger must be called with PageSize+1 (21)
	ledgerRepo.EXPECT().PendingAssignments(gomock.Any(), principal.UserID, 21, 10).
		Return([]ledger.Assignment{}, nil)

	lister := shipment.NewAssignedLister(ledgerRepo, shipmentRepo)
	_, err := lister.List(context.Background(), principal, 10)

	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}
}

func TestHU07_AssignedLister_OrderPreserved(t *testing.T) {
	ctrl := gomock.NewController(t)
	ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)
	shipmentRepo := mocks.NewMockShipmentRepository(ctrl)

	principal := shipment.Principal{UserID: "partner-1", Role: shipment.RoleGroundPartner}

	assignedAt1 := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	assignedAt2 := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	assignedAt3 := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)

	assignments := []ledger.Assignment{
		{ShipmentID: "id-1", UpdateID: 10, AssignedAt: assignedAt1},
		{ShipmentID: "id-2", UpdateID: 20, AssignedAt: assignedAt2},
		{ShipmentID: "id-3", UpdateID: 30, AssignedAt: assignedAt3},
	}

	ledgerRepo.EXPECT().PendingAssignments(gomock.Any(), principal.UserID, 21, 0).
		Return(assignments, nil)

	// Shipments returned in different order from the ledger
	shipments := []shipment.Shipment{
		{ID: "id-3", Code: "HBX-C", Origin: "MIA", Destination: "BOG", Flight: "AV-305", WeightKg: "300", Stage: shipment.StageAssignedToPartner},
		{ID: "id-1", Code: "HBX-A", Origin: "BOG", Destination: "MIA", Flight: "AV-101", WeightKg: "100", Stage: shipment.StageAssignedToPartner},
		{ID: "id-2", Code: "HBX-B", Origin: "BOG", Destination: "MIA", Flight: "AV-202", WeightKg: "200", Stage: shipment.StageAssignedToPartner},
	}

	shipmentRepo.EXPECT().FindByIDs(gomock.Any(), []string{"id-1", "id-2", "id-3"}).
		Return(shipments, nil)

	lister := shipment.NewAssignedLister(ledgerRepo, shipmentRepo)
	page, err := lister.List(context.Background(), principal, 0)

	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}

	// Result must be in ledger order
	want := []shipment.AssignedShipment{
		{Code: "HBX-A", Stage: shipment.StageAssignedToPartner, Origin: "BOG", Destination: "MIA", Flight: "AV-101", WeightKg: "100", AssignedAt: assignedAt1},
		{Code: "HBX-B", Stage: shipment.StageAssignedToPartner, Origin: "BOG", Destination: "MIA", Flight: "AV-202", WeightKg: "200", AssignedAt: assignedAt2},
		{Code: "HBX-C", Stage: shipment.StageAssignedToPartner, Origin: "MIA", Destination: "BOG", Flight: "AV-305", WeightKg: "300", AssignedAt: assignedAt3},
	}

	if !reflect.DeepEqual(page.Shipments, want) {
		t.Errorf("List() shipments = %+v, want %+v", page.Shipments, want)
	}
	if page.Message != nil {
		t.Errorf("List() message = %v, want nil when shipments exist", page.Message)
	}
}

func TestHU07_AssignedLister_NextOffset(t *testing.T) {
	ctrl := gomock.NewController(t)
	ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)
	shipmentRepo := mocks.NewMockShipmentRepository(ctrl)

	principal := shipment.Principal{UserID: "partner-1", Role: shipment.RoleGroundPartner}

	tests := []struct {
		name           string
		assignmentCount int
		offset         int
		wantNextOffset *int
	}{
		{
			name:            "20 assignments, no next page",
			assignmentCount: 20,
			offset:          0,
			wantNextOffset:  nil,
		},
		{
			name:            "21 assignments, next page exists",
			assignmentCount: 21,
			offset:          0,
			wantNextOffset:  intPtr(20),
		},
		{
			name:            "21 assignments from offset 20, next is 40",
			assignmentCount: 21,
			offset:          20,
			wantNextOffset:  intPtr(40),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assignments := make([]ledger.Assignment, tt.assignmentCount)
			shipments := make([]shipment.Shipment, tt.assignmentCount)
			ids := make([]string, tt.assignmentCount)
			for i := 0; i < tt.assignmentCount; i++ {
				id := stringID(i)
				assignments[i] = ledger.Assignment{ShipmentID: id, UpdateID: int64(i), AssignedAt: time.Now()}
				shipments[i] = shipment.Shipment{ID: id, Code: "HBX-" + id, Origin: "BOG", Destination: "MIA", Flight: "AV-100", WeightKg: "100", Stage: shipment.StageAssignedToPartner}
				ids[i] = id
			}

			ledgerRepo.EXPECT().PendingAssignments(gomock.Any(), principal.UserID, 21, tt.offset).
				Return(assignments, nil)

			// Only the first 20 are fetched
			if tt.assignmentCount > 20 {
				shipmentRepo.EXPECT().FindByIDs(gomock.Any(), ids[:20]).Return(shipments[:20], nil)
			} else {
				shipmentRepo.EXPECT().FindByIDs(gomock.Any(), ids).Return(shipments, nil)
			}

			lister := shipment.NewAssignedLister(ledgerRepo, shipmentRepo)
			page, err := lister.List(context.Background(), principal, tt.offset)

			if err != nil {
				t.Fatalf("List() unexpected error: %v", err)
			}

			expectedCount := tt.assignmentCount
			if expectedCount > 20 {
				expectedCount = 20
			}
			if len(page.Shipments) != expectedCount {
				t.Errorf("List() shipments count = %d, want %d", len(page.Shipments), expectedCount)
			}

			if !reflect.DeepEqual(page.NextOffset, tt.wantNextOffset) {
				t.Errorf("List() nextOffset = %v, want %v", page.NextOffset, tt.wantNextOffset)
			}
		})
	}
}

func intPtr(v int) *int {
	return &v
}

func stringID(i int) string {
	return "shipment-" + string(rune('A'+i))
}
