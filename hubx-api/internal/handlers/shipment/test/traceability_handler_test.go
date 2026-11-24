package test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"hubx-api/internal/handlers/shipment"
	"hubx-api/internal/handlers/shipment/mocks"
	domainshipment "hubx-api/internal/shipment"
)

const (
	testCode       = "HBX-TEST0001"
	operatorID     = "11111111-1111-4111-8111-111111111111"
	operatorEmail  = "operator@hubx.co"
	testPrincipalID = "22222222-2222-4222-8222-222222222222"
)

var testPrincipal = domainshipment.Principal{
	UserID: testPrincipalID,
	Role:   domainshipment.RoleCustomer,
}

func TestHU04_TraceabilityHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := mocks.NewMockTraceabilityReader(ctrl)
	principals := mocks.NewMockPrincipalSource(ctrl)

	traceability := &domainshipment.Traceability{
		ShipmentCode: testCode,
		Stage:        "Assigned to partner",
		Updates: []domainshipment.UpdateView{
			{
				Sequence: 1,
				Stage:    "In flight",
				ResponsibleParty: domainshipment.Party{
					ID:   operatorID,
					Name: "Laura Operator",
					Role: "operator",
				},
				AssignedPartner:     nil,
				RecordedAt:          "2026-11-25T10:15:30.123456789Z",
				PreviousFingerprint: nil,
				Fingerprint:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
			{
				Sequence: 2,
				Stage:    "At destination airport",
				ResponsibleParty: domainshipment.Party{
					ID:   operatorID,
					Name: "Laura Operator",
					Role: "operator",
				},
				AssignedPartner: nil,
				RecordedAt:      "2026-11-25T12:00:00.5Z",
				PreviousFingerprint: ptrString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
				Fingerprint:         "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			},
			{
				Sequence: 3,
				Stage:    "Assigned to partner",
				ResponsibleParty: domainshipment.Party{
					ID:   operatorID,
					Name: "Laura Operator",
					Role: "operator",
				},
				AssignedPartner: &domainshipment.Party{
					ID:   "33333333-3333-4333-8333-333333333333",
					Name: "Transportes Rápido",
				},
				RecordedAt:          "2026-11-25T12:30:00Z",
				PreviousFingerprint: ptrString("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
				Fingerprint:         "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			},
		},
	}

	principals.EXPECT().Principal(gomock.Any()).Return(testPrincipal, nil)
	reader.EXPECT().Read(gomock.Any(), testCode, testPrincipal).Return(traceability, nil)

	mux := http.NewServeMux()
	requireAuth := func(h http.Handler) http.Handler { return h }
	shipment.RegisterTraceability(mux, requireAuth, reader, principals)

	req := httptest.NewRequest("GET", "/shipments/"+testCode+"/traceability", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["shipmentCode"] != testCode {
		t.Errorf("shipmentCode = %v, want %q", response["shipmentCode"], testCode)
	}
	if response["stage"] != "Assigned to partner" {
		t.Errorf("stage = %v, want %q", response["stage"], "Assigned to partner")
	}

	updates, ok := response["updates"].([]any)
	if !ok {
		t.Fatalf("updates is not an array: %T", response["updates"])
	}
	if len(updates) != 3 {
		t.Fatalf("updates count = %d, want 3", len(updates))
	}

	// Verify first update structure
	update1 := updates[0].(map[string]any)
	if update1["sequence"].(float64) != 1 {
		t.Errorf("update[0].sequence = %v, want 1", update1["sequence"])
	}
	if update1["stage"] != "In flight" {
		t.Errorf("update[0].stage = %v, want %q", update1["stage"], "In flight")
	}
	responsibleParty := update1["responsibleParty"].(map[string]any)
	if responsibleParty["id"] != operatorID {
		t.Errorf("update[0].responsibleParty.id = %v, want %q", responsibleParty["id"], operatorID)
	}
	if responsibleParty["name"] != "Laura Operator" {
		t.Errorf("update[0].responsibleParty.name = %v, want %q", responsibleParty["name"], "Laura Operator")
	}
	if responsibleParty["role"] != "operator" {
		t.Errorf("update[0].responsibleParty.role = %v, want %q", responsibleParty["role"], "operator")
	}
	if update1["assignedPartner"] != nil {
		t.Errorf("update[0].assignedPartner = %v, want nil", update1["assignedPartner"])
	}
	if update1["previousFingerprint"] != nil {
		t.Errorf("update[0].previousFingerprint = %v, want nil", update1["previousFingerprint"])
	}

	// Verify third update has assigned partner
	update3 := updates[2].(map[string]any)
	assignedPartner := update3["assignedPartner"].(map[string]any)
	if assignedPartner["id"] != "33333333-3333-4333-8333-333333333333" {
		t.Errorf("update[2].assignedPartner.id = %v", assignedPartner["id"])
	}
	if assignedPartner["name"] != "Transportes Rápido" {
		t.Errorf("update[2].assignedPartner.name = %v", assignedPartner["name"])
	}
}

func TestHU04_TraceabilityHandler_EmptyUpdates(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := mocks.NewMockTraceabilityReader(ctrl)
	principals := mocks.NewMockPrincipalSource(ctrl)

	traceability := &domainshipment.Traceability{
		ShipmentCode: testCode,
		Stage:        "Created",
		Updates:      []domainshipment.UpdateView{},
	}

	principals.EXPECT().Principal(gomock.Any()).Return(testPrincipal, nil)
	reader.EXPECT().Read(gomock.Any(), testCode, testPrincipal).Return(traceability, nil)

	mux := http.NewServeMux()
	requireAuth := func(h http.Handler) http.Handler { return h }
	shipment.RegisterTraceability(mux, requireAuth, reader, principals)

	req := httptest.NewRequest("GET", "/shipments/"+testCode+"/traceability", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	updates, ok := response["updates"].([]any)
	if !ok {
		t.Fatalf("updates is not an array: %T", response["updates"])
	}
	if len(updates) != 0 {
		t.Errorf("updates count = %d, want 0", len(updates))
	}
}

func TestHU04_TraceabilityHandler_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := mocks.NewMockTraceabilityReader(ctrl)
	principals := mocks.NewMockPrincipalSource(ctrl)

	principals.EXPECT().Principal(gomock.Any()).Return(testPrincipal, nil)
	reader.EXPECT().Read(gomock.Any(), testCode, testPrincipal).Return(nil, domainshipment.ErrShipmentNotFound)

	mux := http.NewServeMux()
	requireAuth := func(h http.Handler) http.Handler { return h }
	shipment.RegisterTraceability(mux, requireAuth, reader, principals)

	req := httptest.NewRequest("GET", "/shipments/"+testCode+"/traceability", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	errorObj, ok := response["error"].(map[string]any)
	if !ok {
		t.Fatalf("error is not an object: %T", response["error"])
	}
	if errorObj["code"] != "SHIPMENT_NOT_FOUND" {
		t.Errorf("error.code = %v, want %q", errorObj["code"], "SHIPMENT_NOT_FOUND")
	}
	if errorObj["message"] != "No shipment exists with that code." {
		t.Errorf("error.message = %v, want %q", errorObj["message"], "No shipment exists with that code.")
	}
}

func TestHU04_TraceabilityHandler_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := mocks.NewMockTraceabilityReader(ctrl)
	principals := mocks.NewMockPrincipalSource(ctrl)

	principals.EXPECT().Principal(gomock.Any()).Return(domainshipment.Principal{}, errors.New("no session"))

	mux := http.NewServeMux()
	requireAuth := func(h http.Handler) http.Handler { return h }
	shipment.RegisterTraceability(mux, requireAuth, reader, principals)

	req := httptest.NewRequest("GET", "/shipments/"+testCode+"/traceability", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// HU-04 uses the flat 401 body from RequireAuth, not the nested SESSION_EXPIRED
	if response["error"] != "UNAUTHORIZED" {
		t.Errorf("error = %v, want %q", response["error"], "UNAUTHORIZED")
	}
}

func TestHU04_TraceabilityHandler_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := mocks.NewMockTraceabilityReader(ctrl)
	principals := mocks.NewMockPrincipalSource(ctrl)

	principals.EXPECT().Principal(gomock.Any()).Return(testPrincipal, nil)
	reader.EXPECT().Read(gomock.Any(), testCode, testPrincipal).Return(nil, errors.New("database failure"))

	mux := http.NewServeMux()
	requireAuth := func(h http.Handler) http.Handler { return h }
	shipment.RegisterTraceability(mux, requireAuth, reader, principals)

	req := httptest.NewRequest("GET", "/shipments/"+testCode+"/traceability", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	errorObj, ok := response["error"].(map[string]any)
	if !ok {
		t.Fatalf("error is not an object: %T", response["error"])
	}
	if errorObj["code"] != "INTERNAL_ERROR" {
		t.Errorf("error.code = %v, want %q", errorObj["code"], "INTERNAL_ERROR")
	}
	if errorObj["message"] != "The traceability could not be retrieved." {
		t.Errorf("error.message = %v", errorObj["message"])
	}

	// Verify no fingerprint or token in logs (logs go to test output but we can't see them here)
	body := w.Body.String()
	if strings.Contains(body, "aaaa") || strings.Contains(body, "Bearer") {
		t.Error("response body contains sensitive data")
	}
}

func ptrString(s string) *string {
	return &s
}
