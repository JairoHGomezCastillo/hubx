package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"hubx-api/internal/handlers/shipment"
	"hubx-api/internal/handlers/shipment/mocks"
	"hubx-api/internal/ledger"
	domainshipment "hubx-api/internal/shipment"
)

const (
	handlerShipmentCode = "HBX-SEED-0001"
	handlerShipmentID   = "11111111-1111-4111-8111-111111111111"
	handlerOperatorID   = "22222222-2222-4222-8222-222222222222"
	handlerPartnerID    = "33333333-3333-4333-8333-333333333333"
	handlerFingerprint  = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

var (
	errHandlerLookup  = errors.New("directory unavailable")
	errNoPrincipal    = errors.New("no authenticated principal")
	handlerOtherID    = "44444444-4444-4444-8444-444444444444"
	handlerPartnerOne = "Transportes Rápido"
	handlerPartnerTwo = "Logística Express"
)

func handlerPtr(value string) *string { return &value }

// passthroughAuth keeps the handler tests focused on HU-03 behavior; the
// HU-01 RequireAuth middleware is verified by its own package.
func passthroughAuth(next http.Handler) http.Handler { return next }

type responseParty struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type responseBody struct {
	ID                  int64          `json:"id"`
	ShipmentCode        string         `json:"shipmentCode"`
	Sequence            int            `json:"sequence"`
	Stage               string         `json:"stage"`
	ResponsibleParty    responseParty  `json:"responsibleParty"`
	AssignedPartner     *responseParty `json:"assignedPartner"`
	RecordedAt          time.Time      `json:"recordedAt"`
	PreviousFingerprint *string        `json:"previousFingerprint"`
	Fingerprint         string         `json:"fingerprint"`
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestHU03_RecordUpdateResolvesPartyNames(t *testing.T) {
	recordedAt := time.Date(2026, 11, 24, 10, 15, 30, 0, time.UTC)
	tests := []struct {
		name                string
		stage               domainshipment.Stage
		requestPartnerID    *string
		assignedPartnerID   *string
		responsibleName     string
		responsibleErr      error
		partnerName         string
		partnerErr          error
		wantResponsibleName string
		wantPartnerName     *string
	}{
		{
			name:                "201 resolves the operator name",
			stage:               domainshipment.StageInFlight,
			responsibleName:     "Test Operator",
			wantResponsibleName: "Test Operator",
		},
		{
			name:                "201 resolves both names for an assignment",
			stage:               domainshipment.StageAssignedToPartner,
			requestPartnerID:    handlerPtr(handlerPartnerID),
			assignedPartnerID:   handlerPtr(handlerPartnerID),
			responsibleName:     "Test Operator",
			partnerName:         "Test Partner",
			wantResponsibleName: "Test Operator",
			wantPartnerName:     handlerPtr("Test Partner"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			recorder := mocks.NewMockRecorder(ctrl)
			principals := mocks.NewMockPrincipalSource(ctrl)
			directory := mocks.NewMockPartyDirectory(ctrl)
			partners := mocks.NewMockGroundPartnerRepository(ctrl)

			principal := domainshipment.Principal{UserID: handlerOperatorID, Role: domainshipment.RoleOperator}
			update := &ledger.Update{
				ID:                 7,
				ShipmentID:         handlerShipmentID,
				Sequence:           1,
				Stage:              string(tt.stage),
				ResponsiblePartyID: handlerOperatorID,
				AssignedPartnerID:  tt.assignedPartnerID,
				RecordedAt:         recordedAt,
				Fingerprint:        handlerFingerprint,
			}

			principals.EXPECT().Principal(gomock.Any()).Return(principal, nil)
			recorder.EXPECT().Record(gomock.Any(), handlerShipmentCode, tt.stage, tt.requestPartnerID, principal).Return(update, nil)
			directory.EXPECT().Name(gomock.Any(), handlerOperatorID).Return(tt.responsibleName, tt.responsibleErr)
			if tt.assignedPartnerID != nil {
				directory.EXPECT().Name(gomock.Any(), *tt.assignedPartnerID).Return(tt.partnerName, tt.partnerErr)
			}

			mux := http.NewServeMux()
			shipment.RegisterUpdates(mux, passthroughAuth, recorder, principals, directory, partners)

			requestBody := fmt.Sprintf(`{"stage":%q`, tt.stage)
			if tt.requestPartnerID != nil {
				requestBody += fmt.Sprintf(`,"assignedPartnerId":%q`, *tt.requestPartnerID)
			}
			requestBody += "}"
			req := httptest.NewRequest(http.MethodPost, "/shipments/"+handlerShipmentCode+"/updates", strings.NewReader(requestBody))
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, req)

			if res.Code != http.StatusCreated {
				t.Fatalf("status = %d, want %d (body %s)", res.Code, http.StatusCreated, res.Body.String())
			}
			if got := res.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			var got responseBody
			if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
				t.Fatalf("decode 201 body: %v", err)
			}
			if got.ID != update.ID || got.ShipmentCode != handlerShipmentCode || got.Sequence != update.Sequence {
				t.Errorf("identity fields = (%d, %q, %d), want (%d, %q, %d)",
					got.ID, got.ShipmentCode, got.Sequence, update.ID, handlerShipmentCode, update.Sequence)
			}
			if got.Stage != string(tt.stage) || got.Fingerprint != handlerFingerprint {
				t.Errorf("stage/fingerprint = (%q, %q), want (%q, %q)",
					got.Stage, got.Fingerprint, tt.stage, handlerFingerprint)
			}
			if !got.RecordedAt.Equal(recordedAt) {
				t.Errorf("recordedAt = %v, want %v", got.RecordedAt, recordedAt)
			}
			if got.ResponsibleParty.Name != tt.wantResponsibleName {
				t.Errorf("responsibleParty.name = %q, want %q", got.ResponsibleParty.Name, tt.wantResponsibleName)
			}
			if tt.wantPartnerName == nil {
				if got.AssignedPartner != nil {
					t.Errorf("assignedPartner = %+v, want null", got.AssignedPartner)
				}
				return
			}
			if got.AssignedPartner == nil {
				t.Fatal("assignedPartner = null, want the resolved partner")
			}
			if got.AssignedPartner.Name != *tt.wantPartnerName {
				t.Errorf("assignedPartner.name = %q, want %q", got.AssignedPartner.Name, *tt.wantPartnerName)
			}
		})
	}
}

func TestHU03_RecordUpdateFallsBackToIDsWhenNamesCannotBeResolved(t *testing.T) {
	ctrl := gomock.NewController(t)
	recorder := mocks.NewMockRecorder(ctrl)
	principals := mocks.NewMockPrincipalSource(ctrl)
	directory := mocks.NewMockPartyDirectory(ctrl)
	partners := mocks.NewMockGroundPartnerRepository(ctrl)

	principal := domainshipment.Principal{UserID: handlerOperatorID, Role: domainshipment.RoleOperator}
	update := &ledger.Update{
		ID:                 8,
		ShipmentID:         handlerShipmentID,
		Sequence:           3,
		Stage:              string(domainshipment.StageAssignedToPartner),
		ResponsiblePartyID: handlerOperatorID,
		AssignedPartnerID:  handlerPtr(handlerPartnerID),
		RecordedAt:         time.Date(2026, 11, 24, 10, 15, 30, 0, time.UTC),
		Fingerprint:        handlerFingerprint,
	}

	principals.EXPECT().Principal(gomock.Any()).Return(principal, nil)
	recorder.EXPECT().Record(gomock.Any(), handlerShipmentCode, domainshipment.StageAssignedToPartner,
		handlerPtr(handlerPartnerID), principal).Return(update, nil)
	directory.EXPECT().Name(gomock.Any(), handlerOperatorID).Return("", errHandlerLookup)
	directory.EXPECT().Name(gomock.Any(), handlerPartnerID).Return("", errHandlerLookup)

	mux := http.NewServeMux()
	shipment.RegisterUpdates(mux, passthroughAuth, recorder, principals, directory, partners)

	body := fmt.Sprintf(`{"stage":%q,"assignedPartnerId":%q}`, domainshipment.StageAssignedToPartner, handlerPartnerID)
	req := httptest.NewRequest(http.MethodPost, "/shipments/"+handlerShipmentCode+"/updates", strings.NewReader(body))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body %s)", res.Code, http.StatusCreated, res.Body.String())
	}
	var got responseBody
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode 201 body: %v", err)
	}
	if got.ResponsibleParty.Name != handlerOperatorID {
		t.Errorf("responsibleParty.name = %q, want the ID fallback %q", got.ResponsibleParty.Name, handlerOperatorID)
	}
	if got.AssignedPartner == nil {
		t.Fatal("assignedPartner = null, want the partner with an ID fallback")
	}
	if got.AssignedPartner.Name != handlerPartnerID {
		t.Errorf("assignedPartner.name = %q, want the ID fallback %q", got.AssignedPartner.Name, handlerPartnerID)
	}
}

func TestHU03_ListGroundPartners(t *testing.T) {
	tests := []struct {
		name          string
		role          domainshipment.Role
		principalErr  error
		list          []domainshipment.GroundPartner
		listErr       error
		wantStatus    int
		wantPartners  []responseParty
		wantEmptyJSON bool
		wantErrorCode string
	}{
		{
			name: "200 lists the registered partners in repository order",
			role: domainshipment.RoleOperator,
			list: []domainshipment.GroundPartner{
				{ID: handlerPartnerID, Name: handlerPartnerOne},
				{ID: handlerOtherID, Name: handlerPartnerTwo},
			},
			wantStatus: http.StatusOK,
			wantPartners: []responseParty{
				{ID: handlerPartnerID, Name: handlerPartnerOne},
				{ID: handlerOtherID, Name: handlerPartnerTwo},
			},
		},
		{
			name:       "200 is available to a ground partner",
			role:       domainshipment.RoleGroundPartner,
			list:       []domainshipment.GroundPartner{{ID: handlerPartnerID, Name: handlerPartnerOne}},
			wantStatus: http.StatusOK,
			wantPartners: []responseParty{
				{ID: handlerPartnerID, Name: handlerPartnerOne},
			},
		},
		{
			name:       "200 is available to a customer",
			role:       "customer",
			list:       []domainshipment.GroundPartner{{ID: handlerPartnerID, Name: handlerPartnerOne}},
			wantStatus: http.StatusOK,
			wantPartners: []responseParty{
				{ID: handlerPartnerID, Name: handlerPartnerOne},
			},
		},
		{
			name:          "200 returns an empty array when no partner is registered",
			role:          domainshipment.RoleOperator,
			wantStatus:    http.StatusOK,
			wantEmptyJSON: true,
		},
		{
			name:          "401 without a valid principal",
			principalErr:  errNoPrincipal,
			wantStatus:    http.StatusUnauthorized,
			wantErrorCode: "SESSION_EXPIRED",
		},
		{
			name:          "500 when the partner repository fails",
			role:          domainshipment.RoleOperator,
			listErr:       errHandlerLookup,
			wantStatus:    http.StatusInternalServerError,
			wantErrorCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			recorder := mocks.NewMockRecorder(ctrl)
			principals := mocks.NewMockPrincipalSource(ctrl)
			directory := mocks.NewMockPartyDirectory(ctrl)
			partners := mocks.NewMockGroundPartnerRepository(ctrl)

			principal := domainshipment.Principal{UserID: handlerOperatorID, Role: tt.role}
			if tt.principalErr != nil {
				principal = domainshipment.Principal{}
			}
			principals.EXPECT().Principal(gomock.Any()).Return(principal, tt.principalErr)
			if tt.principalErr == nil {
				partners.EXPECT().List(gomock.Any()).Return(tt.list, tt.listErr)
			}

			mux := http.NewServeMux()
			shipment.RegisterUpdates(mux, passthroughAuth, recorder, principals, directory, partners)

			req := httptest.NewRequest(http.MethodGet, "/ground-partners", nil)
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, req)

			if res.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", res.Code, tt.wantStatus, res.Body.String())
			}
			if got := res.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			if tt.wantErrorCode != "" {
				var got errorBody
				if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
					t.Fatalf("decode error body: %v", err)
				}
				if got.Error.Code != tt.wantErrorCode {
					t.Errorf("error.code = %q, want %q", got.Error.Code, tt.wantErrorCode)
				}
				return
			}
			if tt.wantEmptyJSON {
				if got := res.Body.String(); got != "[]" {
					t.Errorf("body = %q, want []", got)
				}
				return
			}
			var got []responseParty
			if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
				t.Fatalf("decode 200 body: %v", err)
			}
			if !reflect.DeepEqual(got, tt.wantPartners) {
				t.Errorf("partners = %+v, want %+v", got, tt.wantPartners)
			}
		})
	}
}
