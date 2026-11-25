package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"hubx-api/internal/evidence"
	"hubx-api/internal/evidence/mocks"
	"hubx-api/internal/ledger"
	domainshipment "hubx-api/internal/shipment"
)

var errEvidenceRepository = errors.New("repository failure")

func ptr(value string) *string { return &value }

func sampleShipment() *domainshipment.Shipment {
	return &domainshipment.Shipment{
		ID:           "11111111-1111-4111-8111-111111111111",
		Code:         "HBX-TEST01",
		Origin:       "Bogotá",
		Destination:  "Miami",
		Flight:       "AV-204",
		WeightKg:     "100.0",
		CustomerID:   "22222222-2222-4222-8222-222222222222",
		CustomerName: "Distribuciones Andina",
		Stage:        domainshipment.StageAssignedToPartner,
		CreatedAt:    time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC),
	}
}

func sampleUpdates() []ledger.Update {
	first := time.Date(2026, 11, 21, 10, 15, 30, 123456000, time.UTC)
	second := time.Date(2026, 11, 22, 11, 20, 0, 0, time.UTC)
	return []ledger.Update{
		{
			ID:                 1,
			ShipmentID:         "11111111-1111-4111-8111-111111111111",
			Sequence:           1,
			Stage:              "In flight",
			ResponsiblePartyID: "33333333-3333-4333-8333-333333333333",
			RecordedAt:         first,
			Fingerprint:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		{
			ID:                  2,
			ShipmentID:          "11111111-1111-4111-8111-111111111111",
			Sequence:            2,
			Stage:               "At destination airport",
			ResponsiblePartyID:  "33333333-3333-4333-8333-333333333333",
			AssignedPartnerID:   ptr("44444444-4444-4444-8444-444444444444"),
			RecordedAt:          second,
			PreviousFingerprint: ptr("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			Fingerprint:         "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
	}
}

func canonicalFor(t *testing.T, update ledger.Update) string {
	t.Helper()
	input := ledger.AppendInput{
		ShipmentID:          update.ShipmentID,
		Sequence:            update.Sequence,
		Stage:               update.Stage,
		ResponsiblePartyID:  update.ResponsiblePartyID,
		AssignedPartnerID:   update.AssignedPartnerID,
		RecordedAt:          update.RecordedAt.UTC().Truncate(time.Microsecond),
		PreviousFingerprint: update.PreviousFingerprint,
	}
	return ledger.CanonicalInput(input)
}

func fixedGeneratedAt() time.Time {
	return time.Date(2026, 11, 25, 9, 30, 0, 0, time.UTC)
}

func TestHU10_ReaderAssemblesEvidenceInChainOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	shipments := mocks.NewMockShipmentRepository(ctrl)
	ledgerRepo := mocks.NewMockLedgerRepository(ctrl)
	parties := mocks.NewMockPartyRepository(ctrl)

	principal := domainshipment.Principal{UserID: "33333333-3333-4333-8333-333333333333", Role: domainshipment.RoleOperator}
	updates := sampleUpdates()
	shipments.EXPECT().FindByCode(gomock.Any(), "HBX-TEST01").Return(sampleShipment(), nil)
	ledgerRepo.EXPECT().List(gomock.Any(), "11111111-1111-4111-8111-111111111111").Return(updates, nil)
	parties.EXPECT().Parties(gomock.Any(), gomock.Any()).Return(map[string]domainshipment.Party{
		"33333333-3333-4333-8333-333333333333": {ID: "33333333-3333-4333-8333-333333333333", Name: "Laura Operator", Role: "operator"},
		"44444444-4444-4444-8444-444444444444": {ID: "44444444-4444-4444-8444-444444444444", Name: "Transportes Rápido", Role: "ground partner"},
	}, nil)

	reader := evidence.NewReader(shipments, ledgerRepo, parties,
		evidence.WithClock(func() time.Time { return fixedGeneratedAt() }))

	got, err := reader.Read(context.Background(), "HBX-TEST01", principal)
	if err != nil {
		t.Fatalf("Read() unexpected error: %v", err)
	}
	if got.Shipment.Code != "HBX-TEST01" || got.Shipment.Origin != "Bogotá" ||
		got.Shipment.Destination != "Miami" || got.Shipment.Flight != "AV-204" ||
		got.Shipment.WeightKg != "100.0" || got.Shipment.CustomerName != "Distribuciones Andina" ||
		got.Shipment.Stage != "Assigned to partner" {
		t.Errorf("Read() shipment data = %+v, want the complete projection", got.Shipment)
	}
	if !got.GeneratedAt.Equal(fixedGeneratedAt()) {
		t.Errorf("GeneratedAt = %v, want %v", got.GeneratedAt, fixedGeneratedAt())
	}
	if len(got.Updates) != 2 {
		t.Fatalf("Read() updates = %d, want 2", len(got.Updates))
	}
	wantParties := []string{"Laura Operator", "Laura Operator"}
	wantPrevious := []string{"-", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	for i, update := range got.Updates {
		if update.Sequence != i+1 {
			t.Errorf("update %d sequence = %d, want %d", i, update.Sequence, i+1)
		}
		if update.ResponsibleParty != wantParties[i] {
			t.Errorf("update %d responsible party = %q, want %q", i, update.ResponsibleParty, wantParties[i])
		}
		if update.PreviousFingerprint != wantPrevious[i] {
			t.Errorf("update %d previous fingerprint = %q, want %q", i, update.PreviousFingerprint, wantPrevious[i])
		}
		if update.CanonicalInput != canonicalFor(t, updates[i]) {
			t.Errorf("update %d canonical input = %q, want the HU-03 formula", i, update.CanonicalInput)
		}
	}
	if got.Updates[1].AssignedPartner != "Transportes Rápido" {
		t.Errorf("second update assigned partner = %q, want Transportes Rápido", got.Updates[1].AssignedPartner)
	}
	if got.Instructions == "" {
		t.Error("Instructions are empty, want self-contained verification steps")
	}
}

func TestHU10_ReaderAllowsEmptyChain(t *testing.T) {
	ctrl := gomock.NewController(t)
	shipments := mocks.NewMockShipmentRepository(ctrl)
	ledgerRepo := mocks.NewMockLedgerRepository(ctrl)
	parties := mocks.NewMockPartyRepository(ctrl)

	principal := domainshipment.Principal{UserID: "33333333-3333-4333-8333-333333333333", Role: domainshipment.RoleOperator}
	shipments.EXPECT().FindByCode(gomock.Any(), "HBX-EMPTY").Return(sampleShipment(), nil)
	ledgerRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return([]ledger.Update{}, nil)
	parties.EXPECT().Parties(gomock.Any(), []string{}).Return(map[string]domainshipment.Party{}, nil)

	got, err := evidence.NewReader(shipments, ledgerRepo, parties).Read(context.Background(), "HBX-EMPTY", principal)
	if err != nil {
		t.Fatalf("Read() unexpected error: %v", err)
	}
	if len(got.Updates) != 0 {
		t.Errorf("Read() updates = %d, want 0", len(got.Updates))
	}
}

func TestHU10_ReaderRefusesRolesAndPreservesPrivacy(t *testing.T) {
	tests := []struct {
		name          string
		principal     domainshipment.Principal
		stored        *domainshipment.Shipment
		wantErr       error
		wantFindCalls int
	}{
		{
			name:          "ground partner cannot download evidence",
			principal:     domainshipment.Principal{UserID: "44444444-4444-4444-8444-444444444444", Role: domainshipment.RoleGroundPartner},
			stored:        sampleShipment(),
			wantErr:       evidence.ErrForbidden,
			wantFindCalls: 0,
		},
		{
			name:          "customer cannot download evidence",
			principal:     domainshipment.Principal{UserID: "22222222-2222-4222-8222-222222222222", Role: domainshipment.RoleCustomer},
			stored:        sampleShipment(),
			wantErr:       evidence.ErrForbidden,
			wantFindCalls: 0,
		},
		{
			name:          "unknown code and invisible shipment share one error",
			principal:     domainshipment.Principal{UserID: "33333333-3333-4333-8333-333333333333", Role: domainshipment.RoleOperator},
			stored:        nil,
			wantErr:       evidence.ErrShipmentNotFound,
			wantFindCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			shipments := mocks.NewMockShipmentRepository(ctrl)
			ledgerRepo := mocks.NewMockLedgerRepository(ctrl)
			parties := mocks.NewMockPartyRepository(ctrl)
			shipments.EXPECT().FindByCode(gomock.Any(), gomock.Any()).Times(tt.wantFindCalls).Return(tt.stored, nil)

			got, err := evidence.NewReader(shipments, ledgerRepo, parties).Read(context.Background(), "HBX-TEST01", tt.principal)
			if got != nil {
				t.Errorf("Read() = %+v, want nil on refusal", got)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Read() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestHU10_ReaderPropagatesRepositoryFailures(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*mocks.MockShipmentRepository, *mocks.MockLedgerRepository, *mocks.MockPartyRepository)
	}{
		{
			name: "shipment lookup failure",
			setup: func(s *mocks.MockShipmentRepository, _ *mocks.MockLedgerRepository, _ *mocks.MockPartyRepository) {
				s.EXPECT().FindByCode(gomock.Any(), gomock.Any()).Return(nil, errEvidenceRepository)
			},
		},
		{
			name: "ledger failure",
			setup: func(s *mocks.MockShipmentRepository, l *mocks.MockLedgerRepository, _ *mocks.MockPartyRepository) {
				s.EXPECT().FindByCode(gomock.Any(), gomock.Any()).Return(sampleShipment(), nil)
				l.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errEvidenceRepository)
			},
		},
		{
			name: "party lookup failure",
			setup: func(s *mocks.MockShipmentRepository, l *mocks.MockLedgerRepository, p *mocks.MockPartyRepository) {
				s.EXPECT().FindByCode(gomock.Any(), gomock.Any()).Return(sampleShipment(), nil)
				l.EXPECT().List(gomock.Any(), gomock.Any()).Return(sampleUpdates(), nil)
				p.EXPECT().Parties(gomock.Any(), gomock.Any()).Return(nil, errEvidenceRepository)
			},
		},
	}

	principal := domainshipment.Principal{UserID: "33333333-3333-4333-8333-333333333333", Role: domainshipment.RoleOperator}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			shipments := mocks.NewMockShipmentRepository(ctrl)
			ledgerRepo := mocks.NewMockLedgerRepository(ctrl)
			parties := mocks.NewMockPartyRepository(ctrl)
			tt.setup(shipments, ledgerRepo, parties)

			got, err := evidence.NewReader(shipments, ledgerRepo, parties).Read(context.Background(), "HBX-TEST01", principal)
			if got != nil {
				t.Errorf("Read() = %+v, want nil on repository failure", got)
			}
			if !errors.Is(err, errEvidenceRepository) {
				t.Errorf("Read() error = %v, want wrapped repository failure", err)
			}
			if errors.Is(err, evidence.ErrShipmentNotFound) {
				t.Error("repository failure must not be reported as SHIPMENT_NOT_FOUND")
			}
		})
	}
}
