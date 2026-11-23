package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockService implements Service for testing.
type mockService struct {
	signInFunc func(ctx context.Context, email, password string) (string, *User, error)
}

func (m *mockService) SignIn(ctx context.Context, email, password string) (string, *User, error) {
	if m.signInFunc != nil {
		return m.signInFunc(ctx, email, password)
	}
	return "", nil, errors.New("not implemented")
}

func TestHandler_SignIn_Success(t *testing.T) {
	service := &mockService{
		signInFunc: func(ctx context.Context, email, password string) (string, *User, error) {
			return "test-jwt-token", &User{
				Email: "operator@hubx.co",
				Role:  "operator",
			}, nil
		},
	}

	handler := NewHandler(service)

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

func TestHandler_SignIn_MissingPassword(t *testing.T) {
	service := &mockService{}
	handler := NewHandler(service)

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

func TestHandler_SignIn_MissingEmail(t *testing.T) {
	service := &mockService{}
	handler := NewHandler(service)

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

func TestHandler_SignIn_BothFieldsMissing(t *testing.T) {
	service := &mockService{}
	handler := NewHandler(service)

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

func TestHandler_SignIn_InvalidCredentials(t *testing.T) {
	service := &mockService{
		signInFunc: func(ctx context.Context, email, password string) (string, *User, error) {
			return "", nil, ErrInvalidCredentials
		},
	}

	handler := NewHandler(service)

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

func TestHandler_SignIn_InternalError(t *testing.T) {
	service := &mockService{
		signInFunc: func(ctx context.Context, email, password string) (string, *User, error) {
			return "", nil, errors.New("database connection failed")
		},
	}

	handler := NewHandler(service)

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

func TestHandler_SignIn_MalformedJSON(t *testing.T) {
	service := &mockService{}
	handler := NewHandler(service)

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

func TestHandler_SignIn_GroundPartnerRole(t *testing.T) {
	service := &mockService{
		signInFunc: func(ctx context.Context, email, password string) (string, *User, error) {
			return "test-jwt-token", &User{
				Email: "rapido@hubx.co",
				Role:  "ground partner", // With space
			}, nil
		},
	}

	handler := NewHandler(service)

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
