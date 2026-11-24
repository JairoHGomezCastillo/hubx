package test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	domain "hubx-api/internal/auth"
	"hubx-api/internal/auth/mocks"
	"hubx-api/internal/handlers/auth"
)

type signInResponse struct {
	Token string   `json:"token"`
	User  userInfo `json:"user"`
}

type userInfo struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func TestHU01_Handler_SignIn_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := mocks.NewMockService(ctrl)
	service.EXPECT().
		SignIn(gomock.Any(), "operator@hubx.co", "test-password").
		Return("test-jwt-token", &domain.User{
			Email: "operator@hubx.co",
			Role:  "operator",
		}, nil)

	handler := auth.NewHandler(service)

	body := `{"email":"operator@hubx.co","password":"test-password"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handler.SignIn(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d; want %d", w.Code, http.StatusOK)
	}

	var resp signInResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Token != "test-jwt-token" {
		t.Errorf("response.Token = %q; want %q", resp.Token, "test-jwt-token")
	}
	if resp.User.Email != "operator@hubx.co" {
		t.Errorf("response.User.Email = %q; want %q", resp.User.Email, "operator@hubx.co")
	}
	if resp.User.Role != "operator" {
		t.Errorf("response.User.Role = %q; want %q", resp.User.Role, "operator")
	}
}

func TestHU01_Handler_SignIn_MissingPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := mocks.NewMockService(ctrl)
	handler := auth.NewHandler(service)

	body := `{"email":"operator@hubx.co"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handler.SignIn(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d; want %d", w.Code, http.StatusBadRequest)
	}

	var resp errorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != "MISSING_REQUIRED_FIELDS" {
		t.Errorf("response.Error = %q; want %q", resp.Error, "MISSING_REQUIRED_FIELDS")
	}
	if resp.Message != "Email and password are required" {
		t.Errorf("response.Message = %q; want %q", resp.Message, "Email and password are required")
	}
}

func TestHU01_Handler_SignIn_MissingEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := mocks.NewMockService(ctrl)
	handler := auth.NewHandler(service)

	body := `{"password":"test-password"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handler.SignIn(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d; want %d", w.Code, http.StatusBadRequest)
	}

	var resp errorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != "MISSING_REQUIRED_FIELDS" {
		t.Errorf("response.Error = %q; want %q", resp.Error, "MISSING_REQUIRED_FIELDS")
	}
}

func TestHU01_Handler_SignIn_BothFieldsMissing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := mocks.NewMockService(ctrl)
	handler := auth.NewHandler(service)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handler.SignIn(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d; want %d", w.Code, http.StatusBadRequest)
	}

	var resp errorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != "MISSING_REQUIRED_FIELDS" {
		t.Errorf("response.Error = %q; want %q", resp.Error, "MISSING_REQUIRED_FIELDS")
	}
}

func TestHU01_Handler_SignIn_InvalidCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := mocks.NewMockService(ctrl)
	service.EXPECT().
		SignIn(gomock.Any(), "operator@hubx.co", "wrong-password").
		Return("", nil, domain.ErrInvalidCredentials)

	handler := auth.NewHandler(service)

	body := `{"email":"operator@hubx.co","password":"wrong-password"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handler.SignIn(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d; want %d", w.Code, http.StatusUnauthorized)
	}

	var resp errorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != "INVALID_CREDENTIALS" {
		t.Errorf("response.Error = %q; want %q", resp.Error, "INVALID_CREDENTIALS")
	}
	if resp.Message != "Invalid credentials" {
		t.Errorf("response.Message = %q; want %q", resp.Message, "Invalid credentials")
	}
}

func TestHU01_Handler_SignIn_InternalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := mocks.NewMockService(ctrl)
	service.EXPECT().
		SignIn(gomock.Any(), "operator@hubx.co", "test-password").
		Return("", nil, errors.New("database connection failed"))

	handler := auth.NewHandler(service)

	body := `{"email":"operator@hubx.co","password":"test-password"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handler.SignIn(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d; want %d", w.Code, http.StatusInternalServerError)
	}

	var resp errorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != "INTERNAL_SERVER_ERROR" {
		t.Errorf("response.Error = %q; want %q", resp.Error, "INTERNAL_SERVER_ERROR")
	}
}

func TestHU01_Handler_SignIn_MalformedJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := mocks.NewMockService(ctrl)
	handler := auth.NewHandler(service)

	body := `{"email":"operator@hubx.co","password":` // Invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handler.SignIn(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d; want %d", w.Code, http.StatusBadRequest)
	}

	var resp errorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != "MISSING_REQUIRED_FIELDS" {
		t.Errorf("response.Error = %q; want %q", resp.Error, "MISSING_REQUIRED_FIELDS")
	}
}

func TestHU01_Handler_SignIn_GroundPartnerRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := mocks.NewMockService(ctrl)
	service.EXPECT().
		SignIn(gomock.Any(), "rapido@hubx.co", "test-password").
		Return("test-jwt-token", &domain.User{
			Email: "rapido@hubx.co",
			Role:  "ground partner", // With space
		}, nil)

	handler := auth.NewHandler(service)

	body := `{"email":"rapido@hubx.co","password":"test-password"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/sign-in", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	handler.SignIn(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d; want %d", w.Code, http.StatusOK)
	}

	var resp signInResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.User.Role != "ground partner" {
		t.Errorf("response.User.Role = %q; want %q", resp.User.Role, "ground partner")
	}
}
