package test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	domainauth "hubx-api/internal/auth"
	handlerauth "hubx-api/internal/handlers/auth"
	"hubx-api/internal/handlers/shipment"
	"hubx-api/internal/handlers/shipment/mocks"
	domainshipment "hubx-api/internal/shipment"

	"go.uber.org/mock/gomock"
)

// mockRequireAuth simulates the HU-01 RequireAuth middleware by injecting
// claims into the context for testing.
func mockRequireAuth(principal domainshipment.Principal) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := &domainauth.Claims{
				Sub:  principal.UserID,
				Role: string(principal.Role),
			}
			ctx := context.WithValue(r.Context(), handlerauth.ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TestHU05_IntegrityHandlerReturns200ForIntactChain verifies the happy path:
// an operator verifies an intact chain and receives a 200 JSON response.
func TestHU05_IntegrityHandlerReturns200ForIntactChain(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	verifier := mocks.NewMockVerifier(ctrl)

	principal := domainshipment.Principal{UserID: "user-1", Role: domainshipment.RoleOperator}
	verification := &domainshipment.Verification{
		ShipmentCode:       "HBX-TEST01",
		Intact:             true,
		UpdatesChecked:     3,
		FirstAlteredUpdate: 0,
	}

	verifier.EXPECT().Verify(gomock.Any(), "HBX-TEST01", principal).Return(verification, nil)

	mux := http.NewServeMux()
	requireAuth := mockRequireAuth(principal)
	shipment.RegisterIntegrity(mux, requireAuth, verifier)

	req := httptest.NewRequest("GET", "/shipments/HBX-TEST01/integrity", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["shipmentCode"] != "HBX-TEST01" {
		t.Errorf("shipmentCode = %v, want HBX-TEST01", response["shipmentCode"])
	}
	if response["intact"] != true {
		t.Errorf("intact = %v, want true", response["intact"])
	}
	if response["updatesChecked"] != float64(3) {
		t.Errorf("updatesChecked = %v, want 3", response["updatesChecked"])
	}
	if response["firstAlteredUpdate"] != nil {
		t.Errorf("firstAlteredUpdate = %v, want nil", response["firstAlteredUpdate"])
	}
}

// TestHU05_IntegrityHandlerReturns200ForAlteredChain verifies that an altered
// chain returns 200 with intact=false and the position.
func TestHU05_IntegrityHandlerReturns200ForAlteredChain(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	verifier := mocks.NewMockVerifier(ctrl)

	principal := domainshipment.Principal{UserID: "user-1", Role: domainshipment.RoleOperator}
	verification := &domainshipment.Verification{
		ShipmentCode:       "HBX-TEST01",
		Intact:             false,
		UpdatesChecked:     3,
		FirstAlteredUpdate: 2,
	}

	verifier.EXPECT().Verify(gomock.Any(), "HBX-TEST01", principal).Return(verification, nil)

	mux := http.NewServeMux()
	requireAuth := mockRequireAuth(principal)
	shipment.RegisterIntegrity(mux, requireAuth, verifier)

	req := httptest.NewRequest("GET", "/shipments/HBX-TEST01/integrity", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["intact"] != false {
		t.Errorf("intact = %v, want false", response["intact"])
	}
	if response["firstAlteredUpdate"] != float64(2) {
		t.Errorf("firstAlteredUpdate = %v, want 2", response["firstAlteredUpdate"])
	}
}

// TestHU05_IntegrityHandlerReturns404ForUnknownShipment verifies that an
// unknown shipment code returns 404 with the SHIPMENT_NOT_FOUND error.
func TestHU05_IntegrityHandlerReturns404ForUnknownShipment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	verifier := mocks.NewMockVerifier(ctrl)

	principal := domainshipment.Principal{UserID: "user-1", Role: domainshipment.RoleOperator}

	verifier.EXPECT().Verify(gomock.Any(), "HBX-UNKNOWN", principal).Return(nil, domainshipment.ErrShipmentNotFound)

	mux := http.NewServeMux()
	requireAuth := mockRequireAuth(principal)
	shipment.RegisterIntegrity(mux, requireAuth, verifier)

	req := httptest.NewRequest("GET", "/shipments/HBX-UNKNOWN/integrity", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["error"] != "SHIPMENT_NOT_FOUND" {
		t.Errorf("error = %v, want SHIPMENT_NOT_FOUND", response["error"])
	}
	if response["message"] != "No shipment exists with that code" {
		t.Errorf("message = %v, want 'No shipment exists with that code'", response["message"])
	}
}

// TestHU05_IntegrityHandlerReturns401WhenNotAuthenticated verifies that a
// missing or invalid JWT returns 401 via RequireAuth.
func TestHU05_IntegrityHandlerReturns401WhenNotAuthenticated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	verifier := mocks.NewMockVerifier(ctrl)

	mux := http.NewServeMux()
	requireAuth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate RequireAuth rejecting the request
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"UNAUTHORIZED","message":"Unauthorized"}`))
		})
	}
	shipment.RegisterIntegrity(mux, requireAuth, verifier)

	req := httptest.NewRequest("GET", "/shipments/HBX-TEST01/integrity", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["error"] != "UNAUTHORIZED" {
		t.Errorf("error = %v, want UNAUTHORIZED", response["error"])
	}
}

// TestHU05_IntegrityHandlerReturns500OnInternalError verifies that an
// unexpected error returns 500.
func TestHU05_IntegrityHandlerReturns500OnInternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	verifier := mocks.NewMockVerifier(ctrl)

	principal := domainshipment.Principal{UserID: "user-1", Role: domainshipment.RoleOperator}

	verifier.EXPECT().Verify(gomock.Any(), "HBX-TEST01", principal).Return(nil, errors.New("database error"))

	mux := http.NewServeMux()
	requireAuth := mockRequireAuth(principal)
	shipment.RegisterIntegrity(mux, requireAuth, verifier)

	req := httptest.NewRequest("GET", "/shipments/HBX-TEST01/integrity", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["error"] != "INTERNAL_SERVER_ERROR" {
		t.Errorf("error = %v, want INTERNAL_SERVER_ERROR", response["error"])
	}
}
