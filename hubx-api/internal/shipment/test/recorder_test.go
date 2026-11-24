package test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"hubx-api/internal/ledger"
	ledgermocks "hubx-api/internal/ledger/mocks"
	"hubx-api/internal/shipment"
	shipmentmocks "hubx-api/internal/shipment/mocks"
)

const (
	recorderShipmentCode = "HBX-SEED-0001"
	recorderShipmentID   = "11111111-1111-4111-8111-111111111111"
	recorderOperatorID   = "22222222-2222-4222-8222-222222222222"
	recorderPartnerID    = "33333333-3333-4333-8333-333333333333"
	recorderOtherPartner = "44444444-4444-4444-8444-444444444444"
	recorderCustomerID   = "55555555-5555-4555-8555-555555555555"
	recorderFingerprint  = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

var errRecorderLookup = errors.New("shipment lookup failed")

func recorderPtr(value string) *string { return &value }

func recorderPrincipal(userID string, role shipment.Role) shipment.Principal {
	return shipment.Principal{UserID: userID, Role: role}
}

func recorderShipment(current shipment.Stage, partnerID *string) *shipment.Shipment {
	return &shipment.Shipment{
		ID:                recorderShipmentID,
		Code:              recorderShipmentCode,
		Stage:             current,
		CustomerID:        recorderCustomerID,
		AssignedPartnerID: partnerID,
		CreatedBy:         recorderOperatorID,
	}
}

func TestHU03_RecordAdvancesTheStageAndChainsTheUpdate(t *testing.T) {
	tests := []struct {
		name             string
		current          shipment.Stage
		requested        shipment.Stage
		principal        shipment.Principal
		assignedPartner  *string
		partnerID        *string
		registered       bool
		last             *ledger.Update
		wantSequence     int
		wantPrevious     *string
		wantAssigned     *string
		wantPartnerCheck bool
	}{
		{
			name:         "Created to In flight starts the chain",
			current:      shipment.StageCreated,
			requested:    shipment.StageInFlight,
			principal:    recorderPrincipal(recorderOperatorID, shipment.RoleOperator),
			wantSequence: 1,
		},
		{
			name:         "In flight to At destination airport links to the last fingerprint",
			current:      shipment.StageInFlight,
			requested:    shipment.StageAtDestinationAirport,
			principal:    recorderPrincipal(recorderOperatorID, shipment.RoleOperator),
			last:         &ledger.Update{Sequence: 1, Fingerprint: recorderFingerprint},
			wantSequence: 2,
			wantPrevious: recorderPtr(recorderFingerprint),
		},
		{
			name:             "At destination airport to Assigned to partner names a registered partner",
			current:          shipment.StageAtDestinationAirport,
			requested:        shipment.StageAssignedToPartner,
			principal:        recorderPrincipal(recorderOperatorID, shipment.RoleOperator),
			partnerID:        recorderPtr(recorderPartnerID),
			registered:       true,
			last:             &ledger.Update{Sequence: 2, Fingerprint: recorderFingerprint},
			wantSequence:     3,
			wantPrevious:     recorderPtr(recorderFingerprint),
			wantAssigned:     recorderPtr(recorderPartnerID),
			wantPartnerCheck: true,
		},
		{
			name:            "Assigned to partner to Out for delivery is recorded by the assigned partner",
			current:         shipment.StageAssignedToPartner,
			requested:       shipment.StageOutForDelivery,
			principal:       recorderPrincipal(recorderPartnerID, shipment.RoleGroundPartner),
			assignedPartner: recorderPtr(recorderPartnerID),
			last:            &ledger.Update{Sequence: 3, Fingerprint: recorderFingerprint},
			wantSequence:    4,
			wantPrevious:    recorderPtr(recorderFingerprint),
		},
		{
			name:            "Out for delivery to Delivered is recorded by the assigned partner",
			current:         shipment.StageOutForDelivery,
			requested:       shipment.StageDelivered,
			principal:       recorderPrincipal(recorderPartnerID, shipment.RoleGroundPartner),
			assignedPartner: recorderPtr(recorderPartnerID),
			last:            &ledger.Update{Sequence: 4, Fingerprint: recorderFingerprint},
			wantSequence:    5,
			wantPrevious:    recorderPtr(recorderFingerprint),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			shipments := shipmentmocks.NewMockShipmentRepository(ctrl)
			partners := shipmentmocks.NewMockGroundPartnerRepository(ctrl)
			ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)
			recorder := shipment.NewRecorder(shipments, partners, ledgerRepo)

			shipments.EXPECT().FindByCode(gomock.Any(), recorderShipmentCode).
				Return(recorderShipment(tt.current, tt.assignedPartner), nil)
			if tt.wantPartnerCheck {
				partners.EXPECT().IsRegisteredPartner(gomock.Any(), recorderPartnerID).
					Return(tt.registered, nil)
			}
			ledgerRepo.EXPECT().Last(gomock.Any(), recorderShipmentID).Return(tt.last, nil)

			var appended *ledger.Update
			ledgerRepo.EXPECT().Append(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, in ledger.AppendInput) (*ledger.Update, error) {
					if in.ShipmentID != recorderShipmentID {
						t.Errorf("shipment id = %q, want %q", in.ShipmentID, recorderShipmentID)
					}
					if in.Sequence != tt.wantSequence {
						t.Errorf("sequence = %d, want %d", in.Sequence, tt.wantSequence)
					}
					if in.Stage != string(tt.requested) {
						t.Errorf("stage = %q, want %q", in.Stage, tt.requested)
					}
					if in.ResponsiblePartyID != tt.principal.UserID {
						t.Errorf("responsible party = %q, want %q", in.ResponsiblePartyID, tt.principal.UserID)
					}
					if in.RecordedAt.IsZero() || in.RecordedAt.Location() != time.UTC {
						t.Errorf("recorded date = %v, want a non-zero UTC date", in.RecordedAt)
					}
					if (in.PreviousFingerprint == nil) != (tt.wantPrevious == nil) {
						t.Fatalf("previous fingerprint = %v, want %v", in.PreviousFingerprint, tt.wantPrevious)
					}
					if tt.wantPrevious != nil && *in.PreviousFingerprint != *tt.wantPrevious {
						t.Errorf("previous fingerprint = %q, want %q", *in.PreviousFingerprint, *tt.wantPrevious)
					}
					if (in.AssignedPartnerID == nil) != (tt.wantAssigned == nil) {
						t.Fatalf("assigned partner = %v, want %v", in.AssignedPartnerID, tt.wantAssigned)
					}
					if tt.wantAssigned != nil && *in.AssignedPartnerID != *tt.wantAssigned {
						t.Errorf("assigned partner = %q, want %q", *in.AssignedPartnerID, *tt.wantAssigned)
					}
					if err := ledger.ValidateAppendInput(in); err != nil {
						t.Errorf("ValidateAppendInput() = %v, want nil", err)
					}
					if err := ledger.CheckSequence(in.Sequence, tt.last); err != nil {
						t.Errorf("CheckSequence() = %v, want nil", err)
					}
					if err := ledger.CheckPredecessor(in, tt.last); err != nil {
						t.Errorf("CheckPredecessor() = %v, want nil", err)
					}

					appended = &ledger.Update{
						Sequence:            in.Sequence,
						Stage:               in.Stage,
						ResponsiblePartyID:  in.ResponsiblePartyID,
						AssignedPartnerID:   in.AssignedPartnerID,
						RecordedAt:          in.RecordedAt,
						PreviousFingerprint: in.PreviousFingerprint,
						Fingerprint:         strings.Repeat("b", 64),
					}
					return appended, nil
				},
			).Times(1)

			got, err := recorder.Record(context.Background(), recorderShipmentCode, tt.requested, tt.partnerID, tt.principal)
			if err != nil {
				t.Fatalf("Record() unexpected error: %v", err)
			}
			if got == nil || got != appended {
				t.Fatalf("Record() = %+v, want the appended update %+v", got, appended)
			}
			if got.Stage != string(tt.requested) || got.Sequence != tt.wantSequence {
				t.Errorf("update = %+v, want stage %q at sequence %d", got, tt.requested, tt.wantSequence)
			}
		})
	}
}

func TestHU03_RecordRejectsAttemptsThatBreakTheRules(t *testing.T) {
	operator := recorderPrincipal(recorderOperatorID, shipment.RoleOperator)
	assignedPartner := recorderPrincipal(recorderPartnerID, shipment.RoleGroundPartner)

	tests := []struct {
		name            string
		current         shipment.Stage
		requested       shipment.Stage
		principal       shipment.Principal
		stored          *shipment.Shipment
		unknown         bool
		findErr         error
		partnerID       *string
		registered      bool
		wantErr         error
		wantPartnerCall bool
	}{
		{
			name:      "Created to At destination airport skips a stage",
			current:   shipment.StageCreated,
			requested: shipment.StageAtDestinationAirport,
			principal: operator,
			wantErr:   shipment.ErrOutOfSequence,
		},
		{
			name:      "In flight cannot be repeated",
			current:   shipment.StageInFlight,
			requested: shipment.StageInFlight,
			principal: operator,
			wantErr:   shipment.ErrOutOfSequence,
		},
		{
			name:      "Assigned to partner cannot be replaced",
			current:   shipment.StageAssignedToPartner,
			requested: shipment.StageAssignedToPartner,
			principal: operator,
			stored:    recorderShipment(shipment.StageAssignedToPartner, recorderPtr(recorderPartnerID)),
			wantErr:   shipment.ErrOutOfSequence,
		},
		{
			name:      "Delivered is terminal",
			current:   shipment.StageDelivered,
			requested: shipment.StageInFlight,
			principal: operator,
			stored:    recorderShipment(shipment.StageDelivered, recorderPtr(recorderPartnerID)),
			wantErr:   shipment.ErrOutOfSequence,
		},
		{
			name:      "an unknown stage is not an update",
			current:   shipment.StageCreated,
			requested: shipment.Stage("Unknown"),
			principal: operator,
			wantErr:   shipment.ErrOutOfSequence,
		},
		{
			name:      "a customer cannot record an early update",
			current:   shipment.StageCreated,
			requested: shipment.StageInFlight,
			principal: recorderPrincipal(recorderCustomerID, shipment.RoleCustomer),
			wantErr:   shipment.ErrNotResponsible,
		},
		{
			name:      "a ground partner cannot record an early update",
			current:   shipment.StageInFlight,
			requested: shipment.StageAtDestinationAirport,
			principal: recorderPrincipal(recorderPartnerID, shipment.RoleGroundPartner),
			wantErr:   shipment.ErrNotResponsible,
		},
		{
			name:      "the operator cannot record delivery",
			current:   shipment.StageOutForDelivery,
			requested: shipment.StageDelivered,
			principal: operator,
			stored:    recorderShipment(shipment.StageOutForDelivery, recorderPtr(recorderPartnerID)),
			wantErr:   shipment.ErrNotResponsible,
		},
		{
			name:      "another ground partner cannot record delivery",
			current:   shipment.StageAssignedToPartner,
			requested: shipment.StageOutForDelivery,
			principal: recorderPrincipal(recorderOtherPartner, shipment.RoleGroundPartner),
			stored:    recorderShipment(shipment.StageAssignedToPartner, recorderPtr(recorderPartnerID)),
			wantErr:   shipment.ErrNotResponsible,
		},
		{
			name:      "an unknown code is not responsible",
			current:   shipment.StageCreated,
			requested: shipment.StageInFlight,
			principal: operator,
			unknown:   true,
			wantErr:   shipment.ErrNotResponsible,
		},
		{
			name:      "a shipment the customer cannot see is not responsible",
			current:   shipment.StageCreated,
			requested: shipment.StageInFlight,
			principal: recorderPrincipal(recorderOtherPartner, shipment.RoleCustomer),
			stored:    recorderShipment(shipment.StageCreated, nil),
			wantErr:   shipment.ErrNotResponsible,
		},
		{
			name:      "a shipment the partner is not assigned to is not responsible",
			current:   shipment.StageAssignedToPartner,
			requested: shipment.StageOutForDelivery,
			principal: assignedPartner,
			stored:    recorderShipment(shipment.StageAssignedToPartner, recorderPtr(recorderOtherPartner)),
			wantErr:   shipment.ErrNotResponsible,
		},
		{
			name:      "a lookup failure is indistinguishable from an unknown code",
			current:   shipment.StageCreated,
			requested: shipment.StageInFlight,
			principal: operator,
			findErr:   errRecorderLookup,
			wantErr:   shipment.ErrNotResponsible,
		},
		{
			name:      "assignment without a partner id is rejected",
			current:   shipment.StageAtDestinationAirport,
			requested: shipment.StageAssignedToPartner,
			principal: operator,
			wantErr:   shipment.ErrPartnerRequired,
		},
		{
			name:      "assignment with an empty partner id is rejected",
			current:   shipment.StageAtDestinationAirport,
			requested: shipment.StageAssignedToPartner,
			principal: operator,
			partnerID: recorderPtr(""),
			wantErr:   shipment.ErrPartnerRequired,
		},
		{
			name:            "a customer cannot be named as delivery partner",
			current:         shipment.StageAtDestinationAirport,
			requested:       shipment.StageAssignedToPartner,
			principal:       operator,
			partnerID:       recorderPtr(recorderCustomerID),
			registered:      false,
			wantErr:         shipment.ErrNotGroundPartner,
			wantPartnerCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			shipments := shipmentmocks.NewMockShipmentRepository(ctrl)
			partners := shipmentmocks.NewMockGroundPartnerRepository(ctrl)
			ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)
			recorder := shipment.NewRecorder(shipments, partners, ledgerRepo)

			stored := tt.stored
			if stored == nil && !tt.unknown && tt.findErr == nil {
				stored = recorderShipment(tt.current, nil)
			}
			shipments.EXPECT().FindByCode(gomock.Any(), recorderShipmentCode).Return(stored, tt.findErr)
			if tt.wantPartnerCall {
				partners.EXPECT().IsRegisteredPartner(gomock.Any(), *tt.partnerID).Return(tt.registered, nil)
			}
			// No Last or Append expectation: a rejected attempt must not touch the ledger.

			got, err := recorder.Record(context.Background(), recorderShipmentCode, tt.requested, tt.partnerID, tt.principal)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if got != nil {
				t.Errorf("update = %+v, want nil on rejection", got)
			}
		})
	}
}

func TestHU03_UnknownAndInvisibleCodesGiveTheSameAnswer(t *testing.T) {
	otherCustomer := recorderPrincipal(recorderOtherPartner, shipment.RoleCustomer)
	cases := []struct {
		name   string
		stored *shipment.Shipment
		err    error
	}{
		{name: "unknown code"},
		{name: "invisible shipment", stored: recorderShipment(shipment.StageCreated, nil)},
		{name: "lookup failure", err: errRecorderLookup},
	}

	var firstMessage string
	for i, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			shipments := shipmentmocks.NewMockShipmentRepository(ctrl)
			partners := shipmentmocks.NewMockGroundPartnerRepository(ctrl)
			ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)
			recorder := shipment.NewRecorder(shipments, partners, ledgerRepo)
			shipments.EXPECT().FindByCode(gomock.Any(), recorderShipmentCode).Return(tt.stored, tt.err)

			_, err := recorder.Record(context.Background(), recorderShipmentCode, shipment.StageInFlight, nil, otherCustomer)
			if !errors.Is(err, shipment.ErrNotResponsible) {
				t.Fatalf("error = %v, want %v", err, shipment.ErrNotResponsible)
			}
			if i == 0 {
				firstMessage = err.Error()
			} else if err.Error() != firstMessage {
				t.Errorf("error message = %q, want the indistinguishable %q", err.Error(), firstMessage)
			}
		})
	}
}

func TestHU03_RecordPropagatesPersistenceFailures(t *testing.T) {
	tests := []struct {
		name      string
		lastErr   error
		appendErr error
		wantErr   error
	}{
		{name: "last lookup failure", lastErr: errRecorderLookup, wantErr: errRecorderLookup},
		{name: "append failure", appendErr: shipment.ErrConcurrentUpdate, wantErr: shipment.ErrConcurrentUpdate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			shipments := shipmentmocks.NewMockShipmentRepository(ctrl)
			partners := shipmentmocks.NewMockGroundPartnerRepository(ctrl)
			ledgerRepo := ledgermocks.NewMockLedgerRepository(ctrl)
			recorder := shipment.NewRecorder(shipments, partners, ledgerRepo)

			shipments.EXPECT().FindByCode(gomock.Any(), recorderShipmentCode).
				Return(recorderShipment(shipment.StageCreated, nil), nil)
			ledgerRepo.EXPECT().Last(gomock.Any(), recorderShipmentID).Return(nil, tt.lastErr)
			if tt.lastErr == nil {
				ledgerRepo.EXPECT().Append(gomock.Any(), gomock.Any()).Return(nil, tt.appendErr)
			}

			got, err := recorder.Record(context.Background(), recorderShipmentCode, shipment.StageInFlight, nil,
				recorderPrincipal(recorderOperatorID, shipment.RoleOperator))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if got != nil {
				t.Errorf("update = %+v, want nil on persistence failure", got)
			}
		})
	}
}
