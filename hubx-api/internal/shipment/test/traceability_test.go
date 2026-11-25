package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"hubx-api/internal/ledger"
	"hubx-api/internal/shipment"
	"hubx-api/internal/shipment/mocks"
)

const (
	operatorID      = "11111111-1111-4111-8111-111111111111"
	customerID      = "22222222-2222-4222-8222-222222222222"
	partnerID       = "33333333-3333-4333-8333-333333333333"
	otherCustomerID = "44444444-4444-4444-8444-444444444444"
	otherPartnerID  = "55555555-5555-4555-8555-555555555555"
	shipmentID      = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	shipmentCode    = "HBX-TEST0001"
)

var (
	operatorPrincipal = shipment.Principal{
		UserID: operatorID,
		Role:   shipment.RoleOperator,
	}
	customerPrincipal = shipment.Principal{
		UserID: customerID,
		Role:   shipment.RoleCustomer,
	}
	otherCustomerPrincipal = shipment.Principal{
		UserID: otherCustomerID,
		Role:   shipment.RoleCustomer,
	}
	partnerPrincipal = shipment.Principal{
		UserID: partnerID,
		Role:   shipment.RoleGroundPartner,
	}
	unassignedPartnerPrincipal = shipment.Principal{
		UserID: otherPartnerID,
		Role:   shipment.RoleGroundPartner,
	}
)

func TestHU04_TraceabilityReader_Visibility(t *testing.T) {
	assignedPartner := partnerID
	currentShipment := &shipment.Shipment{
		ID:                shipmentID,
		Code:              shipmentCode,
		Stage:             shipment.StageAssignedToPartner,
		CustomerID:        customerID,
		AssignedPartnerID: &assignedPartner,
	}

	tests := []struct {
		name      string
		shipment  *shipment.Shipment
		principal shipment.Principal
		wantErr   error
	}{
		{
			name:      "operator can view any shipment",
			shipment:  currentShipment,
			principal: operatorPrincipal,
			wantErr:   nil,
		},
		{
			name:      "customer can view own shipment",
			shipment:  currentShipment,
			principal: customerPrincipal,
			wantErr:   nil,
		},
		{
			name:      "customer cannot view other's shipment",
			shipment:  currentShipment,
			principal: otherCustomerPrincipal,
			wantErr:   shipment.ErrShipmentNotFound,
		},
		{
			name:      "assigned partner can view shipment",
			shipment:  currentShipment,
			principal: partnerPrincipal,
			wantErr:   nil,
		},
		{
			name: "unassigned partner cannot view shipment",
			shipment: &shipment.Shipment{
				ID:                shipmentID,
				Code:              shipmentCode,
				Stage:             shipment.StageCreated,
				CustomerID:        customerID,
				AssignedPartnerID: nil,
			},
			principal: unassignedPartnerPrincipal,
			wantErr:   shipment.ErrShipmentNotFound,
		},
		{
			name: "partner cannot view shipment assigned to another",
			shipment: &shipment.Shipment{
				ID:                shipmentID,
				Code:              shipmentCode,
				Stage:             shipment.StageAssignedToPartner,
				CustomerID:        customerID,
				AssignedPartnerID: &assignedPartner,
			},
			principal: unassignedPartnerPrincipal,
			wantErr:   shipment.ErrShipmentNotFound,
		},
		{
			name:      "unknown shipment returns not found",
			shipment:  nil,
			principal: operatorPrincipal,
			wantErr:   shipment.ErrShipmentNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			shipments := mocks.NewMockShipmentRepository(ctrl)
			ledgerRepo := mocks.NewMockLedgerRepository(ctrl)
			parties := mocks.NewMockPartyRepository(ctrl)

			shipments.EXPECT().FindByCode(gomock.Any(), shipmentCode).Return(tt.shipment, nil)

			if tt.wantErr == nil {
				// Visible: expect ledger and party calls
				ledgerRepo.EXPECT().List(gomock.Any(), shipmentID).Return([]ledger.Update{}, nil)
				parties.EXPECT().Parties(gomock.Any(), gomock.Any()).Return(map[string]shipment.Party{}, nil)
			}

			reader := shipment.NewTraceabilityReader(shipments, ledgerRepo, parties)
			result, err := reader.Read(context.Background(), shipmentCode, tt.principal)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Read() error = %v, want %v", err, tt.wantErr)
				}
				if result != nil {
					t.Errorf("Read() result = %+v, want nil on error", result)
				}
			} else {
				if err != nil {
					t.Errorf("Read() unexpected error: %v", err)
				}
				if result == nil {
					t.Fatal("Read() result is nil, want non-nil")
				}
			}
		})
	}
}

func TestHU04_TraceabilityReader_EmptyChain(t *testing.T) {
	ctrl := gomock.NewController(t)
	shipments := mocks.NewMockShipmentRepository(ctrl)
	ledgerRepo := mocks.NewMockLedgerRepository(ctrl)
	parties := mocks.NewMockPartyRepository(ctrl)

	currentShipment := &shipment.Shipment{
		ID:         shipmentID,
		Code:       shipmentCode,
		Stage:      shipment.StageCreated,
		CustomerID: customerID,
	}

	shipments.EXPECT().FindByCode(gomock.Any(), shipmentCode).Return(currentShipment, nil)
	ledgerRepo.EXPECT().List(gomock.Any(), shipmentID).Return([]ledger.Update{}, nil)
	parties.EXPECT().Parties(gomock.Any(), []string{}).Return(map[string]shipment.Party{}, nil)

	reader := shipment.NewTraceabilityReader(shipments, ledgerRepo, parties)
	result, err := reader.Read(context.Background(), shipmentCode, operatorPrincipal)

	if err != nil {
		t.Fatalf("Read() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Read() result is nil")
	}
	if result.ShipmentCode != shipmentCode {
		t.Errorf("Read() shipmentCode = %q, want %q", result.ShipmentCode, shipmentCode)
	}
	if result.Stage != string(shipment.StageCreated) {
		t.Errorf("Read() stage = %q, want %q", result.Stage, shipment.StageCreated)
	}
	if result.Updates == nil {
		t.Fatal("Read() updates is nil, want non-nil slice")
	}
	if len(result.Updates) != 0 {
		t.Errorf("Read() updates = %d, want 0", len(result.Updates))
	}
}

func TestHU04_TraceabilityReader_OrderAndPartyResolution(t *testing.T) {
	ctrl := gomock.NewController(t)
	shipments := mocks.NewMockShipmentRepository(ctrl)
	ledgerRepo := mocks.NewMockLedgerRepository(ctrl)
	parties := mocks.NewMockPartyRepository(ctrl)

	recordedAt1 := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)
	recordedAt2 := time.Date(2026, 11, 25, 11, 0, 0, 0, time.UTC)
	recordedAt3 := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	fp1 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	fp2 := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	fp3 := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	assignedPartner := partnerID

	updates := []ledger.Update{
		{
			Sequence:           1,
			Stage:              "In flight",
			ResponsiblePartyID: operatorID,
			RecordedAt:         recordedAt1,
			Fingerprint:        fp1,
		},
		{
			Sequence:            2,
			Stage:               "At destination airport",
			ResponsiblePartyID:  operatorID,
			RecordedAt:          recordedAt2,
			PreviousFingerprint: &fp1,
			Fingerprint:         fp2,
		},
		{
			Sequence:            3,
			Stage:               "Assigned to partner",
			ResponsiblePartyID:  operatorID,
			AssignedPartnerID:   &assignedPartner,
			RecordedAt:          recordedAt3,
			PreviousFingerprint: &fp2,
			Fingerprint:         fp3,
		},
	}

	resolvedParties := map[string]shipment.Party{
		operatorID: {ID: operatorID, Name: "Laura Operator", Role: "operator"},
		partnerID:  {ID: partnerID, Name: "Transportes Rápido", Role: "ground partner"},
	}

	currentShipment := &shipment.Shipment{
		ID:                shipmentID,
		Code:              shipmentCode,
		Stage:             shipment.StageAssignedToPartner,
		CustomerID:        customerID,
		AssignedPartnerID: &assignedPartner,
	}

	shipments.EXPECT().FindByCode(gomock.Any(), shipmentCode).Return(currentShipment, nil)
	ledgerRepo.EXPECT().List(gomock.Any(), shipmentID).Return(updates, nil)

	var capturedIDs []string
	parties.EXPECT().Parties(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, ids []string) (map[string]shipment.Party, error) {
			capturedIDs = ids
			return resolvedParties, nil
		})

	reader := shipment.NewTraceabilityReader(shipments, ledgerRepo, parties)
	result, err := reader.Read(context.Background(), shipmentCode, operatorPrincipal)

	if err != nil {
		t.Fatalf("Read() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Read() result is nil")
	}

	// Verify single Parties call with both operator and partner
	if len(capturedIDs) != 2 {
		t.Errorf("Parties() called with %d ids, want 2", len(capturedIDs))
	}
	idsMap := make(map[string]bool)
	for _, id := range capturedIDs {
		idsMap[id] = true
	}
	if !idsMap[operatorID] || !idsMap[partnerID] {
		t.Errorf("Parties() ids = %v, want [%s, %s]", capturedIDs, operatorID, partnerID)
	}

	// Verify order preserved (oldest first)
	if len(result.Updates) != 3 {
		t.Fatalf("Read() updates count = %d, want 3", len(result.Updates))
	}
	if result.Updates[0].Sequence != 1 || result.Updates[1].Sequence != 2 || result.Updates[2].Sequence != 3 {
		t.Errorf("Read() updates not in order: %d, %d, %d", result.Updates[0].Sequence, result.Updates[1].Sequence, result.Updates[2].Sequence)
	}

	// Verify party resolution
	if result.Updates[0].ResponsibleParty.Name != "Laura Operator" {
		t.Errorf("Update 1 responsible party name = %q, want %q", result.Updates[0].ResponsibleParty.Name, "Laura Operator")
	}
	if result.Updates[2].AssignedPartner == nil {
		t.Fatal("Update 3 assigned partner is nil")
	}
	if result.Updates[2].AssignedPartner.Name != "Transportes Rápido" {
		t.Errorf("Update 3 assigned partner name = %q, want %q", result.Updates[2].AssignedPartner.Name, "Transportes Rápido")
	}
	if result.Updates[0].AssignedPartner != nil {
		t.Errorf("Update 1 assigned partner = %+v, want nil", result.Updates[0].AssignedPartner)
	}
}

func TestHU04_TraceabilityReader_RepositoryErrors(t *testing.T) {
	tests := []struct {
		name         string
		shipmentErr  error
		ledgerErr    error
		partiesErr   error
		wantErrorStr string
	}{
		{
			name:         "shipment lookup error propagates",
			shipmentErr:  errors.New("db connection failed"),
			wantErrorStr: "find shipment",
		},
		{
			name:         "ledger list error propagates",
			ledgerErr:    errors.New("query failed"),
			wantErrorStr: "list updates",
		},
		{
			name:         "parties resolution error propagates",
			partiesErr:   errors.New("party query failed"),
			wantErrorStr: "resolve parties",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			shipments := mocks.NewMockShipmentRepository(ctrl)
			ledgerRepo := mocks.NewMockLedgerRepository(ctrl)
			partiesRepo := mocks.NewMockPartyRepository(ctrl)

			currentShipment := &shipment.Shipment{
				ID:         shipmentID,
				Code:       shipmentCode,
				Stage:      shipment.StageCreated,
				CustomerID: customerID,
			}

			if tt.shipmentErr != nil {
				shipments.EXPECT().FindByCode(gomock.Any(), shipmentCode).Return(nil, tt.shipmentErr)
			} else {
				shipments.EXPECT().FindByCode(gomock.Any(), shipmentCode).Return(currentShipment, nil)
				if tt.ledgerErr != nil {
					ledgerRepo.EXPECT().List(gomock.Any(), shipmentID).Return(nil, tt.ledgerErr)
				} else {
					update := ledger.Update{
						Sequence:           1,
						Stage:              "In flight",
						ResponsiblePartyID: operatorID,
						RecordedAt:         time.Now().UTC(),
						Fingerprint:        "abc",
					}
					ledgerRepo.EXPECT().List(gomock.Any(), shipmentID).Return([]ledger.Update{update}, nil)
					partiesRepo.EXPECT().Parties(gomock.Any(), gomock.Any()).Return(nil, tt.partiesErr)
				}
			}

			reader := shipment.NewTraceabilityReader(shipments, ledgerRepo, partiesRepo)
			result, err := reader.Read(context.Background(), shipmentCode, operatorPrincipal)

			if err == nil {
				t.Fatal("Read() expected error, got nil")
			}
			if result != nil {
				t.Errorf("Read() result = %+v, want nil on error", result)
			}
			if tt.wantErrorStr != "" && !contains(err.Error(), tt.wantErrorStr) {
				t.Errorf("Read() error = %q, want it to contain %q", err.Error(), tt.wantErrorStr)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
