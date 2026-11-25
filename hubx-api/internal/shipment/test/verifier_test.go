package test

import (
	"context"
	"errors"
	"testing"

	"hubx-api/internal/ledger"
	ledgermocks "hubx-api/internal/ledger/mocks"
	"hubx-api/internal/shipment"
	"hubx-api/internal/shipment/mocks"

	"go.uber.org/mock/gomock"
)

// TestHU05_VerifierOperatorSeesAnyShipment verifies that an operator can verify
// any shipment's integrity.
func TestHU05_VerifierOperatorSeesAnyShipment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	shipmentRepo := mocks.NewMockShipmentRepository(ctrl)
	ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)

	ship := &shipment.Shipment{
		ID:         "shipment-1",
		Code:       "HBX-TEST01",
		CustomerID: "customer-1",
	}

	shipmentRepo.EXPECT().FindByCode(gomock.Any(), "HBX-TEST01").Return(ship, nil)

	updates := []ledger.Update{
		{ShipmentID: "shipment-1", Sequence: 1, Stage: "In flight", Fingerprint: "fp1"},
	}
	ledgerRepo.EXPECT().List(gomock.Any(), "shipment-1").Return(updates, nil)

	verifier := shipment.NewVerifier(shipmentRepo, ledgerRepo)
	principal := shipment.Principal{UserID: "operator-1", Role: shipment.RoleOperator}

	result, err := verifier.Verify(context.Background(), "HBX-TEST01", principal)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.ShipmentCode != "HBX-TEST01" {
		t.Errorf("ShipmentCode = %s, want HBX-TEST01", result.ShipmentCode)
	}
}

// TestHU05_VerifierCustomerSeesOwnShipment verifies that a customer can verify
// their own shipment.
func TestHU05_VerifierCustomerSeesOwnShipment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	shipmentRepo := mocks.NewMockShipmentRepository(ctrl)
	ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)

	ship := &shipment.Shipment{
		ID:         "shipment-1",
		Code:       "HBX-TEST01",
		CustomerID: "customer-1",
	}

	shipmentRepo.EXPECT().FindByCode(gomock.Any(), "HBX-TEST01").Return(ship, nil)
	ledgerRepo.EXPECT().List(gomock.Any(), "shipment-1").Return([]ledger.Update{}, nil)

	verifier := shipment.NewVerifier(shipmentRepo, ledgerRepo)
	principal := shipment.Principal{UserID: "customer-1", Role: shipment.RoleCustomer}

	result, err := verifier.Verify(context.Background(), "HBX-TEST01", principal)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
}

// TestHU05_VerifierCustomerCannotSeeOthers verifies that a customer cannot see
// another customer's shipment (returns ErrShipmentNotFound).
func TestHU05_VerifierCustomerCannotSeeOthers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	shipmentRepo := mocks.NewMockShipmentRepository(ctrl)
	ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)

	ship := &shipment.Shipment{
		ID:         "shipment-1",
		Code:       "HBX-TEST01",
		CustomerID: "customer-1",
	}

	shipmentRepo.EXPECT().FindByCode(gomock.Any(), "HBX-TEST01").Return(ship, nil)

	verifier := shipment.NewVerifier(shipmentRepo, ledgerRepo)
	principal := shipment.Principal{UserID: "customer-2", Role: shipment.RoleCustomer}

	result, err := verifier.Verify(context.Background(), "HBX-TEST01", principal)
	if !errors.Is(err, shipment.ErrShipmentNotFound) {
		t.Errorf("error = %v, want ErrShipmentNotFound", err)
	}
	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
}

// TestHU05_VerifierGroundPartnerSeesAssigned verifies that a ground partner
// can verify a shipment assigned to them.
func TestHU05_VerifierGroundPartnerSeesAssigned(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	shipmentRepo := mocks.NewMockShipmentRepository(ctrl)
	ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)

	partner := "partner-1"
	ship := &shipment.Shipment{
		ID:                "shipment-1",
		Code:              "HBX-TEST01",
		CustomerID:        "customer-1",
		AssignedPartnerID: &partner,
	}

	shipmentRepo.EXPECT().FindByCode(gomock.Any(), "HBX-TEST01").Return(ship, nil)
	ledgerRepo.EXPECT().List(gomock.Any(), "shipment-1").Return([]ledger.Update{}, nil)

	verifier := shipment.NewVerifier(shipmentRepo, ledgerRepo)
	principal := shipment.Principal{UserID: "partner-1", Role: shipment.RoleGroundPartner}

	result, err := verifier.Verify(context.Background(), "HBX-TEST01", principal)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
}

// TestHU05_VerifierGroundPartnerCannotSeeUnassigned verifies that a ground
// partner cannot see a shipment not assigned to them.
func TestHU05_VerifierGroundPartnerCannotSeeUnassigned(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	shipmentRepo := mocks.NewMockShipmentRepository(ctrl)
	ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)

	partner := "partner-1"
	ship := &shipment.Shipment{
		ID:                "shipment-1",
		Code:              "HBX-TEST01",
		CustomerID:        "customer-1",
		AssignedPartnerID: &partner,
	}

	shipmentRepo.EXPECT().FindByCode(gomock.Any(), "HBX-TEST01").Return(ship, nil)

	verifier := shipment.NewVerifier(shipmentRepo, ledgerRepo)
	principal := shipment.Principal{UserID: "partner-2", Role: shipment.RoleGroundPartner}

	result, err := verifier.Verify(context.Background(), "HBX-TEST01", principal)
	if !errors.Is(err, shipment.ErrShipmentNotFound) {
		t.Errorf("error = %v, want ErrShipmentNotFound", err)
	}
	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
}

// TestHU05_VerifierUnknownShipment verifies that an unknown code returns
// ErrShipmentNotFound.
func TestHU05_VerifierUnknownShipment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	shipmentRepo := mocks.NewMockShipmentRepository(ctrl)
	ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)

	shipmentRepo.EXPECT().FindByCode(gomock.Any(), "HBX-UNKNOWN").Return(nil, nil)

	verifier := shipment.NewVerifier(shipmentRepo, ledgerRepo)
	principal := shipment.Principal{UserID: "operator-1", Role: shipment.RoleOperator}

	result, err := verifier.Verify(context.Background(), "HBX-UNKNOWN", principal)
	if !errors.Is(err, shipment.ErrShipmentNotFound) {
		t.Errorf("error = %v, want ErrShipmentNotFound", err)
	}
	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
}
