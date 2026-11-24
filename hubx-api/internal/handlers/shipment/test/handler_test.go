package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"hubx-api/internal/auth"
	authHandler "hubx-api/internal/handlers/auth"
	"hubx-api/internal/handlers/shipment"
	domain "hubx-api/internal/shipment"
	"hubx-api/internal/shipment/mocks"
)

func TestHU02_CreateShipment_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)

	now := time.Date(2026, 11, 24, 15, 4, 5, 0, time.UTC)

	mockService.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, input domain.CreateInput) (*domain.Shipment, error) {
			if input.Role != "operator" {
				t.Errorf("service.Create() role = %q; want %q", input.Role, "operator")
			}
			if input.OperatorID != "op-id" {
				t.Errorf("service.Create() operatorID = %q; want %q", input.OperatorID, "op-id")
			}
			return &domain.Shipment{
				Code:          "HBX-7F3A9C21",
				Origin:        "Bogotá",
				Destination:   "Miami",
				Flight:        "AV-204",
				WeightKg:      "120",
				CustomerEmail: "andina@hubx.co",
				CustomerName:  "Distribuciones Andina",
				Stage:         "Created",
				CreatedAt:     now,
			}, nil
		})

	handler := shipment.NewHandler(mockService)

	body := `{"origin":"Bogotá","destination":"Miami","flight":"AV-204","weightKg":120,"customerEmail":"andina@hubx.co"}`
	req := httptest.NewRequest("POST", "/shipments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Inject claims into context (simulating RequireAuth middleware)
	claims := &auth.Claims{
		Sub:   "op-id",
		Email: "operator@hubx.co",
		Role:  "operator",
	}
	ctx := context.WithValue(req.Context(), authHandler.ClaimsContextKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Create() status = %d; want %d. Body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["code"] != "HBX-7F3A9C21" {
		t.Errorf("response code = %q; want %q", resp["code"], "HBX-7F3A9C21")
	}
	if resp["stage"] != "Created" {
		t.Errorf("response stage = %q; want %q", resp["stage"], "Created")
	}
	if resp["createdAt"] != "2026-11-24T15:04:05Z" {
		t.Errorf("response createdAt = %q; want %q", resp["createdAt"], "2026-11-24T15:04:05Z")
	}
}

func TestHU02_CreateShipment_NonOperator_ForbiddenWithoutBodyDecode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	// Service should NOT be called when role check fails

	handler := shipment.NewHandler(mockService)

	body := `{"origin":"Bogotá","destination":"Miami","flight":"AV-204","weightKg":120,"customerEmail":"andina@hubx.co"}`
	req := httptest.NewRequest("POST", "/shipments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Inject ground partner claims
	claims := &auth.Claims{
		Sub:   "partner-id",
		Email: "rapido@hubx.co",
		Role:  "ground partner",
	}
	ctx := context.WithValue(req.Context(), authHandler.ClaimsContextKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Create() status = %d; want %d", w.Code, http.StatusForbidden)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["error"] != "FORBIDDEN" {
		t.Errorf("response error = %q; want %q", resp["error"], "FORBIDDEN")
	}
}

func TestHU02_CreateShipment_Customer_Forbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	handler := shipment.NewHandler(mockService)

	body := `{"origin":"Bogotá","destination":"Miami","flight":"AV-204","weightKg":120,"customerEmail":"andina@hubx.co"}`
	req := httptest.NewRequest("POST", "/shipments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	claims := &auth.Claims{
		Sub:   "cust-id",
		Email: "andina@hubx.co",
		Role:  "customer",
	}
	ctx := context.WithValue(req.Context(), authHandler.ClaimsContextKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Create() status = %d; want %d", w.Code, http.StatusForbidden)
	}
}

func TestHU02_CreateShipment_MissingToken_Unauthorized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	handler := shipment.NewHandler(mockService)

	body := `{"origin":"Bogotá","destination":"Miami","flight":"AV-204","weightKg":120,"customerEmail":"andina@hubx.co"}`
	req := httptest.NewRequest("POST", "/shipments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No claims in context (simulating missing token)

	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Create() status = %d; want %d", w.Code, http.StatusUnauthorized)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["error"] != "UNAUTHORIZED" {
		t.Errorf("response error = %q; want %q", resp["error"], "UNAUTHORIZED")
	}
}

func TestHU02_CreateShipment_MalformedJSON_BadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	handler := shipment.NewHandler(mockService)

	body := `{"origin":"Bogotá","destination":`
	req := httptest.NewRequest("POST", "/shipments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	claims := &auth.Claims{
		Sub:   "op-id",
		Email: "operator@hubx.co",
		Role:  "operator",
	}
	ctx := context.WithValue(req.Context(), authHandler.ClaimsContextKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Create() status = %d; want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["error"] != "MISSING_REQUIRED_FIELDS" {
		t.Errorf("response error = %q; want %q", resp["error"], "MISSING_REQUIRED_FIELDS")
	}
}

func TestHU02_CreateShipment_WeightAsString_BadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	handler := shipment.NewHandler(mockService)

	// Weight as a JSON string instead of a number - contract says JSON number
	body := `{"origin":"Bogotá","destination":"Miami","flight":"AV-204","weightKg":"120","customerEmail":"andina@hubx.co"}`
	req := httptest.NewRequest("POST", "/shipments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	claims := &auth.Claims{
		Sub:   "op-id",
		Email: "operator@hubx.co",
		Role:  "operator",
	}
	ctx := context.WithValue(req.Context(), authHandler.ClaimsContextKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Create() status = %d; want %d. Body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["error"] != "WEIGHT_OUT_OF_RANGE" {
		t.Errorf("response error = %q; want WEIGHT_OUT_OF_RANGE", resp["error"])
	}
}

func TestHU02_CreateShipment_BodyOver16KB_BadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	handler := shipment.NewHandler(mockService)

	// Create a body larger than 16 KiB
	largeOrigin := strings.Repeat("A", 20*1024)
	body := `{"origin":"` + largeOrigin + `","destination":"Miami","flight":"AV-204","weightKg":120,"customerEmail":"andina@hubx.co"}`
	req := httptest.NewRequest("POST", "/shipments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	claims := &auth.Claims{
		Sub:   "op-id",
		Email: "operator@hubx.co",
		Role:  "operator",
	}
	ctx := context.WithValue(req.Context(), authHandler.ClaimsContextKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Create() status = %d; want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHU02_CreateShipment_ServiceErrors(t *testing.T) {
	tests := []struct {
		name          string
		serviceErr    error
		wantStatus    int
		wantErrorCode string
		wantMessage   string
	}{
		{
			name:          "missing data",
			serviceErr:    domain.ErrMissingData,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "MISSING_REQUIRED_FIELDS",
			wantMessage:   "Required data is missing",
		},
		{
			name:          "same origin destination",
			serviceErr:    domain.ErrSameOriginDestination,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "SAME_ORIGIN_DESTINATION",
			wantMessage:   "The origin and the destination must be different",
		},
		{
			name:          "weight out of range",
			serviceErr:    domain.ErrWeightOutOfRange,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "WEIGHT_OUT_OF_RANGE",
			wantMessage:   "The weight must be between 0.1 and 5000 kg",
		},
		{
			name:          "customer not found",
			serviceErr:    domain.ErrCustomerNotFound,
			wantStatus:    http.StatusUnprocessableEntity,
			wantErrorCode: "CUSTOMER_NOT_FOUND",
			wantMessage:   "The customer indicated does not exist",
		},
		{
			name:          "not a customer role",
			serviceErr:    domain.ErrNotACustomer,
			wantStatus:    http.StatusUnprocessableEntity,
			wantErrorCode: "CUSTOMER_NOT_CUSTOMER_ROLE",
			wantMessage:   "The customer indicated does not have the customer role",
		},
		{
			name:          "internal error",
			serviceErr:    errors.New("database connection failed"),
			wantStatus:    http.StatusInternalServerError,
			wantErrorCode: "INTERNAL_SERVER_ERROR",
			wantMessage:   "An unexpected error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockService(ctrl)
			mockService.EXPECT().
				Create(gomock.Any(), gomock.Any()).
				Return(nil, tt.serviceErr)

			handler := shipment.NewHandler(mockService)

			body := `{"origin":"Bogotá","destination":"Miami","flight":"AV-204","weightKg":120,"customerEmail":"andina@hubx.co"}`
			req := httptest.NewRequest("POST", "/shipments", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			claims := &auth.Claims{
				Sub:   "op-id",
				Email: "operator@hubx.co",
				Role:  "operator",
			}
			ctx := context.WithValue(req.Context(), authHandler.ClaimsContextKey, claims)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			handler.Create(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Create() status = %d; want %d", w.Code, tt.wantStatus)
			}

			var resp map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if resp["error"] != tt.wantErrorCode {
				t.Errorf("response error = %q; want %q", resp["error"], tt.wantErrorCode)
			}
			if resp["message"] != tt.wantMessage {
				t.Errorf("response message = %q; want %q", resp["message"], tt.wantMessage)
			}
		})
	}
}

func TestHU02_CreateShipment_WithRequireAuth_ExpiredToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockService(ctrl)
	handler := shipment.NewHandler(mockService)

	// Create an expired token (exp in the past)
	token, err := auth.GenerateToken(&auth.User{
		ID:    "op-id",
		Email: "operator@hubx.co",
		Role:  "operator",
	}, "test-secret", -2) // Negative hours = expired
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Create middleware
	middleware := authHandler.NewMiddleware("test-secret")

	// Wrap handler with RequireAuth
	protectedHandler := middleware.RequireAuth(http.HandlerFunc(handler.Create))

	body := `{"origin":"Bogotá","destination":"Miami","flight":"AV-204","weightKg":120,"customerEmail":"andina@hubx.co"}`
	req := httptest.NewRequest("POST", "/shipments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	protectedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Create() with expired token status = %d; want %d", w.Code, http.StatusUnauthorized)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !strings.Contains(resp["message"].(string), "expired") {
		t.Errorf("response message = %q; want to contain 'expired'", resp["message"])
	}
}
