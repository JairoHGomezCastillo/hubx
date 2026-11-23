package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestMiddleware_RequireAuth_ValidToken(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	middleware := NewMiddleware(secret)

	// Generate a valid token
	user := &User{
		ID:    "550e8400-e29b-41d4-a716-446655440000",
		Email: "operator@hubx.co",
		Role:  "operator",
	}
	token, err := GenerateToken(user, secret, 1)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Handler that checks claims were injected
	handler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r.Context())
		if claims == nil {
			t.Error("claims not found in context")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if claims.Email != user.Email {
			t.Errorf("claims.Email = %q; want %q", claims.Email, user.Email)
		}
		if claims.Role != user.Role {
			t.Errorf("claims.Role = %q; want %q", claims.Role, user.Role)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d; want %d", w.Code, http.StatusOK)
	}
}

func TestMiddleware_RequireAuth_MissingToken(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	middleware := NewMiddleware(secret)

	handler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when token is missing")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
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

func TestMiddleware_RequireAuth_InvalidFormat(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	middleware := NewMiddleware(secret)

	handler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
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

func TestMiddleware_RequireAuth_ExpiredToken(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	middleware := NewMiddleware(secret)

	// Create an expired token
	now := time.Now()
	claims := Claims{
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

	handler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when token is expired")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
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

func TestMiddleware_RequireAuth_InvalidSignature(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	wrongSecret := "wrong-secret-key-32-bytes-long!!"
	middleware := NewMiddleware(secret)

	// Generate a token with wrong secret
	user := &User{
		ID:    "550e8400-e29b-41d4-a716-446655440000",
		Email: "operator@hubx.co",
		Role:  "operator",
	}
	token, err := GenerateToken(user, wrongSecret, 1)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	handler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when signature is invalid")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
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

func TestMiddleware_RequireAuth_MalformedToken(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	middleware := NewMiddleware(secret)

	handler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called when token is malformed")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
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

func TestGetClaims(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	middleware := NewMiddleware(secret)

	user := &User{
		ID:    "550e8400-e29b-41d4-a716-446655440000",
		Email: "operator@hubx.co",
		Role:  "operator",
	}
	token, err := GenerateToken(user, secret, 1)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	handler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r.Context())
		if claims == nil {
			t.Error("GetClaims() returned nil")
			return
		}
		if claims.Sub != user.ID {
			t.Errorf("claims.Sub = %q; want %q", claims.Sub, user.ID)
		}
		if claims.Email != user.Email {
			t.Errorf("claims.Email = %q; want %q", claims.Email, user.Email)
		}
		if claims.Role != user.Role {
			t.Errorf("claims.Role = %q; want %q", claims.Role, user.Role)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d; want %d", w.Code, http.StatusOK)
	}
}
