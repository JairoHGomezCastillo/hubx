package test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"hubx-api/internal/handlers/shipment"
	"hubx-api/internal/handlers/shipment/mocks"
	domainshipment "hubx-api/internal/shipment"
)

func TestHU07_AssignedHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	lister := mocks.NewMockAssignedLister(ctrl)
	principals := mocks.NewMockPrincipalSource(ctrl)

	principal := domainshipment.Principal{UserID: "partner-1", Role: domainshipment.RoleGroundPartner}
	assignedAt := time.Date(2026, 11, 24, 10, 15, 30, 123456000, time.UTC)

	page := domainshipment.AssignedPage{
		Shipments: []domainshipment.AssignedShipment{
			{
				Code:        "HBX-7F3A9C21",
				Stage:       domainshipment.StageAssignedToPartner,
				Origin:      "Bogotá",
				Destination: "Miami",
				Flight:      "AV-204",
				WeightKg:    "120",
				AssignedAt:  assignedAt,
			},
		},
		NextOffset: nil,
		Message:    nil,
	}

	principals.EXPECT().Principal(gomock.Any()).Return(principal, nil)
	lister.EXPECT().List(gomock.Any(), principal, 0).Return(page, nil)

	mux := http.NewServeMux()
	shipment.RegisterAssigned(mux, noAuth, lister, principals)

	req := httptest.NewRequest(http.MethodGet, "/assigned-shipments?offset=0", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	shipments, ok := body["shipments"].([]any)
	if !ok || len(shipments) != 1 {
		t.Fatalf("shipments = %v, want array with 1 item", body["shipments"])
	}

	s := shipments[0].(map[string]any)
	if s["code"] != "HBX-7F3A9C21" {
		t.Errorf("code = %v, want HBX-7F3A9C21", s["code"])
	}
	if s["stage"] != "Assigned to partner" {
		t.Errorf("stage = %v, want 'Assigned to partner'", s["stage"])
	}
	if s["assignedAt"] != "2026-11-24T10:15:30.123456Z" {
		t.Errorf("assignedAt = %v, want RFC3339 UTC", s["assignedAt"])
	}

	if body["nextOffset"] != nil {
		t.Errorf("nextOffset = %v, want nil", body["nextOffset"])
	}
	if body["message"] != nil {
		t.Errorf("message = %v, want nil", body["message"])
	}
}

func TestHU07_AssignedHandler_EmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	lister := mocks.NewMockAssignedLister(ctrl)
	principals := mocks.NewMockPrincipalSource(ctrl)

	principal := domainshipment.Principal{UserID: "partner-1", Role: domainshipment.RoleGroundPartner}
	msg := "You have no assigned shipments"
	page := domainshipment.AssignedPage{
		Shipments:  []domainshipment.AssignedShipment{},
		NextOffset: nil,
		Message:    &msg,
	}

	principals.EXPECT().Principal(gomock.Any()).Return(principal, nil)
	lister.EXPECT().List(gomock.Any(), principal, 0).Return(page, nil)

	mux := http.NewServeMux()
	shipment.RegisterAssigned(mux, noAuth, lister, principals)

	req := httptest.NewRequest(http.MethodGet, "/assigned-shipments", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	shipments, ok := body["shipments"].([]any)
	if !ok || len(shipments) != 0 {
		t.Fatalf("shipments = %v, want empty array", body["shipments"])
	}

	if body["message"] != "You have no assigned shipments" {
		t.Errorf("message = %v, want 'You have no assigned shipments'", body["message"])
	}
}

func TestHU07_AssignedHandler_NextOffset(t *testing.T) {
	ctrl := gomock.NewController(t)
	lister := mocks.NewMockAssignedLister(ctrl)
	principals := mocks.NewMockPrincipalSource(ctrl)

	principal := domainshipment.Principal{UserID: "partner-1", Role: domainshipment.RoleGroundPartner}
	next := 20
	page := domainshipment.AssignedPage{
		Shipments:  []domainshipment.AssignedShipment{},
		NextOffset: &next,
		Message:    nil,
	}

	principals.EXPECT().Principal(gomock.Any()).Return(principal, nil)
	lister.EXPECT().List(gomock.Any(), principal, 0).Return(page, nil)

	mux := http.NewServeMux()
	shipment.RegisterAssigned(mux, noAuth, lister, principals)

	req := httptest.NewRequest(http.MethodGet, "/assigned-shipments", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// JSON numbers decode as float64
	if body["nextOffset"] != float64(20) {
		t.Errorf("nextOffset = %v, want 20", body["nextOffset"])
	}
}

func TestHU07_AssignedHandler_OffsetParsing(t *testing.T) {
	tests := []struct {
		name       string
		queryParam string
		wantOffset int
		wantStatus int
		wantError  string
	}{
		{
			name:       "missing offset defaults to 0",
			queryParam: "",
			wantOffset: 0,
			wantStatus: http.StatusOK,
		},
		{
			name:       "offset=0 is parsed",
			queryParam: "?offset=0",
			wantOffset: 0,
			wantStatus: http.StatusOK,
		},
		{
			name:       "offset=100 is parsed",
			queryParam: "?offset=100",
			wantOffset: 100,
			wantStatus: http.StatusOK,
		},
		{
			name:       "offset=-1 passes to domain as -1",
			queryParam: "?offset=-1",
			wantOffset: -1,
			wantStatus: http.StatusBadRequest,
			wantError:  "INVALID_REQUEST",
		},
		{
			name:       "offset=abc is not digits, passed as -1",
			queryParam: "?offset=abc",
			wantOffset: -1,
			wantStatus: http.StatusBadRequest,
			wantError:  "INVALID_REQUEST",
		},
		{
			name:       "offset with decimal is not digits, passed as -1",
			queryParam: "?offset=10.5",
			wantOffset: -1,
			wantStatus: http.StatusBadRequest,
			wantError:  "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			lister := mocks.NewMockAssignedLister(ctrl)
			principals := mocks.NewMockPrincipalSource(ctrl)

			principal := domainshipment.Principal{UserID: "partner-1", Role: domainshipment.RoleGroundPartner}
			principals.EXPECT().Principal(gomock.Any()).Return(principal, nil)

			if tt.wantStatus == http.StatusOK {
				lister.EXPECT().List(gomock.Any(), principal, tt.wantOffset).
					Return(domainshipment.AssignedPage{Shipments: []domainshipment.AssignedShipment{}}, nil)
			} else {
				lister.EXPECT().List(gomock.Any(), principal, tt.wantOffset).
					Return(domainshipment.AssignedPage{}, domainshipment.ErrInvalidOffset)
			}

			mux := http.NewServeMux()
			shipment.RegisterAssigned(mux, noAuth, lister, principals)

			req := httptest.NewRequest(http.MethodGet, "/assigned-shipments"+tt.queryParam, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantError != "" {
				var body map[string]any
				if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
					t.Fatalf("decode error response: %v", err)
				}
				if body["error"] != tt.wantError {
					t.Errorf("error code = %v, want %s", body["error"], tt.wantError)
				}
			}
		})
	}
}

func TestHU07_AssignedHandler_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	lister := mocks.NewMockAssignedLister(ctrl)
	principals := mocks.NewMockPrincipalSource(ctrl)

	tests := []struct {
		name string
		role domainshipment.Role
	}{
		{name: "operator", role: domainshipment.RoleOperator},
		{name: "customer", role: domainshipment.RoleCustomer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			principal := domainshipment.Principal{UserID: "user-1", Role: tt.role}
			principals.EXPECT().Principal(gomock.Any()).Return(principal, nil)
			lister.EXPECT().List(gomock.Any(), principal, 0).
				Return(domainshipment.AssignedPage{}, domainshipment.ErrNoPermission)

			mux := http.NewServeMux()
			shipment.RegisterAssigned(mux, noAuth, lister, principals)

			req := httptest.NewRequest(http.MethodGet, "/assigned-shipments", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
			}

			var body map[string]any
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}

			if body["error"] != "FORBIDDEN" {
				t.Errorf("error = %v, want FORBIDDEN", body["error"])
			}
			if body["message"] != "You do not have permission to see that information" {
				t.Errorf("message = %v, want permission message", body["message"])
			}
		})
	}
}

func TestHU07_AssignedHandler_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	lister := mocks.NewMockAssignedLister(ctrl)
	principals := mocks.NewMockPrincipalSource(ctrl)

	principals.EXPECT().Principal(gomock.Any()).Return(domainshipment.Principal{}, errors.New("missing token"))

	mux := http.NewServeMux()
	shipment.RegisterAssigned(mux, noAuth, lister, principals)

	req := httptest.NewRequest(http.MethodGet, "/assigned-shipments", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if body["error"] != "UNAUTHORIZED" {
		t.Errorf("error = %v, want UNAUTHORIZED", body["error"])
	}
}

func TestHU07_AssignedHandler_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	lister := mocks.NewMockAssignedLister(ctrl)
	principals := mocks.NewMockPrincipalSource(ctrl)

	principal := domainshipment.Principal{UserID: "partner-1", Role: domainshipment.RoleGroundPartner}
	principals.EXPECT().Principal(gomock.Any()).Return(principal, nil)
	lister.EXPECT().List(gomock.Any(), principal, 0).
		Return(domainshipment.AssignedPage{}, errors.New("database unavailable"))

	mux := http.NewServeMux()
	shipment.RegisterAssigned(mux, noAuth, lister, principals)

	req := httptest.NewRequest(http.MethodGet, "/assigned-shipments", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if body["error"] != "INTERNAL_SERVER_ERROR" {
		t.Errorf("error = %v, want INTERNAL_SERVER_ERROR", body["error"])
	}
	if body["message"] != "An unexpected error occurred" {
		t.Errorf("message = %v, want generic message", body["message"])
	}
}

func TestHU07_AssignedHandler_RoleCheckedBeforeOffset(t *testing.T) {
	ctrl := gomock.NewController(t)
	lister := mocks.NewMockAssignedLister(ctrl)
	principals := mocks.NewMockPrincipalSource(ctrl)

	// Customer with bad offset: expect ErrNoPermission, not ErrInvalidOffset
	principal := domainshipment.Principal{UserID: "customer-1", Role: domainshipment.RoleCustomer}
	principals.EXPECT().Principal(gomock.Any()).Return(principal, nil)
	lister.EXPECT().List(gomock.Any(), principal, -1).
		Return(domainshipment.AssignedPage{}, domainshipment.ErrNoPermission)

	mux := http.NewServeMux()
	shipment.RegisterAssigned(mux, noAuth, lister, principals)

	req := httptest.NewRequest(http.MethodGet, "/assigned-shipments?offset=-1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (role checked before offset)", w.Code, http.StatusForbidden)
	}
}

func noAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
