package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"hubx-gw/internal/middleware"
)

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// generateTestToken creates a valid JWT for testing.
func generateTestToken(secret string, expirationHours int) (string, error) {
	now := time.Now()
	claims := middleware.Claims{
		Sub:   "550e8400-e29b-41d4-a716-446655440000",
		Email: "operator@hubx.co",
		Role:  "operator",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expirationHours) * time.Hour)),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func TestHU01_JWTValidation_ValidToken(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	token, err := generateTestToken(secret, 1)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	excludedPaths := map[string]bool{
		"/health": true,
	}

	middlewareFunc := middleware.JWTValidation(secret, excludedPaths)
	
	handler := middlewareFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/shipments", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d; want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "success" {
		t.Errorf("body = %q; want %q", w.Body.String(), "success")
	}
}

func TestHU01_JWTValidation_ExcludedPath(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"

	excludedPaths := map[string]bool{
		"/health":           true,
		"/api/health":       true,
		"/api/auth/sign-in": true,
	}

	middlewareFunc := middleware.JWTValidation(secret, excludedPaths)
	
	handler := middlewareFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	tests := []struct {
		name string
		path string
	}{
		{
			name: "health endpoint",
			path: "/health",
		},
		{
			name: "API health endpoint",
			path: "/api/health",
		},
		{
			name: "sign-in endpoint",
			path: "/api/auth/sign-in",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			// No Authorization header
			
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("status = %d; want %d (path %s should be excluded)", w.Code, http.StatusOK, tt.path)
			}
		})
	}
}

func TestHU01_JWTValidation_MissingToken(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	excludedPaths := map[string]bool{}

	middlewareFunc := middleware.JWTValidation(secret, excludedPaths)
	
	handler := middlewareFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when token is missing")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/shipments", nil)
	// No Authorization header
	
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d; want %d", w.Code, http.StatusUnauthorized)
	}

	var resp errorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != "UNAUTHORIZED" {
		t.Errorf("response.Error = %q; want %q", resp.Error, "UNAUTHORIZED")
	}
	if resp.Message != "Authorization header required" {
		t.Errorf("response.Message = %q; want %q", resp.Message, "Authorization header required")
	}
}

func TestHU01_JWTValidation_ExpiredToken(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	excludedPaths := map[string]bool{}

	// Create an expired token
	now := time.Now()
	claims := middleware.Claims{
		Sub:   "550e8400-e29b-41d4-a716-446655440000",
		Email: "operator@hubx.co",
		Role:  "operator",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)), // Expired 1 hour ago
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to create expired token: %v", err)
	}

	middlewareFunc := middleware.JWTValidation(secret, excludedPaths)
	
	handler := middlewareFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when token is expired")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/shipments", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d; want %d", w.Code, http.StatusUnauthorized)
	}

	var resp errorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != "UNAUTHORIZED" {
		t.Errorf("response.Error = %q; want %q", resp.Error, "UNAUTHORIZED")
	}
	if resp.Message != "Session expired. Please sign in again." {
		t.Errorf("response.Message = %q; want %q", resp.Message, "Session expired. Please sign in again.")
	}
}

func TestHU01_JWTValidation_InvalidSignature(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	wrongSecret := "wrong-secret-key-32-bytes-long!!"
	excludedPaths := map[string]bool{}

	// Generate token with wrong secret
	token, err := generateTestToken(wrongSecret, 1)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	middlewareFunc := middleware.JWTValidation(secret, excludedPaths)
	
	handler := middlewareFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when signature is invalid")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/shipments", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d; want %d", w.Code, http.StatusUnauthorized)
	}

	var resp errorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != "UNAUTHORIZED" {
		t.Errorf("response.Error = %q; want %q", resp.Error, "UNAUTHORIZED")
	}
	if resp.Message != "Invalid token" {
		t.Errorf("response.Message = %q; want %q", resp.Message, "Invalid token")
	}
}

func TestHU01_JWTValidation_MalformedToken(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	excludedPaths := map[string]bool{}

	middlewareFunc := middleware.JWTValidation(secret, excludedPaths)
	
	handler := middlewareFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when token is malformed")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/shipments", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-jwt")
	
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d; want %d", w.Code, http.StatusUnauthorized)
	}

	var resp errorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Error != "UNAUTHORIZED" {
		t.Errorf("response.Error = %q; want %q", resp.Error, "UNAUTHORIZED")
	}
	if resp.Message != "Invalid token" {
		t.Errorf("response.Message = %q; want %q", resp.Message, "Invalid token")
	}
}

func TestHU01_JWTValidation_InvalidAuthorizationFormat(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	excludedPaths := map[string]bool{}

	middlewareFunc := middleware.JWTValidation(secret, excludedPaths)
	
	handler := middlewareFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when authorization format is invalid")
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "no Bearer prefix",
			header: "some-token",
		},
		{
			name:   "wrong scheme",
			header: "Basic some-token",
		},
		{
			name:   "just Bearer",
			header: "Bearer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/shipments", nil)
			req.Header.Set("Authorization", tt.header)
			
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d; want %d", w.Code, http.StatusUnauthorized)
			}

			var resp errorResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if resp.Error != "UNAUTHORIZED" {
				t.Errorf("response.Error = %q; want %q", resp.Error, "UNAUTHORIZED")
			}
		})
	}
}
