package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	domain "hubx-api/internal/auth"
	"hubx-api/internal/handlers/auth"
)

type sessionErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// TestHU03_Middleware_SessionAuth covers the HU-03 session error contract:
// the update routes must report a nested SESSION_EXPIRED body for every kind
// of rejected JWT, while RequireAuth keeps its flat HU-01 body.
func TestHU03_Middleware_SessionAuth(t *testing.T) {
	const secret = "test-secret-key-32-bytes-long!!"

	validToken, err := domain.GenerateToken(&domain.User{
		ID:    "550e8400-e29b-41d4-a716-446655440000",
		Email: "operator@hubx.co",
		Role:  "operator",
	}, secret, 1)
	if err != nil {
		t.Fatalf("generate valid token: %v", err)
	}

	wrongSecretToken, err := domain.GenerateToken(&domain.User{
		ID:    "550e8400-e29b-41d4-a716-446655440000",
		Email: "operator@hubx.co",
		Role:  "operator",
	}, "wrong-secret-key-32-bytes-long!!", 1)
	if err != nil {
		t.Fatalf("generate token with wrong secret: %v", err)
	}

	now := time.Now()
	expiredToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, domain.Claims{
		Sub:   "550e8400-e29b-41d4-a716-446655440000",
		Email: "operator@hubx.co",
		Role:  "operator",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Hour)),
		},
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("generate expired token: %v", err)
	}

	tests := []struct {
		name      string
		header    string
		wantValid bool
	}{
		{name: "valid token", header: "Bearer " + validToken, wantValid: true},
		{name: "missing Authorization header", header: ""},
		{name: "wrong scheme", header: "Basic " + validToken},
		{name: "missing Bearer value", header: "Bearer"},
		{name: "malformed JWT", header: "Bearer not-a-valid-jwt"},
		{name: "invalid signature", header: "Bearer " + wrongSecretToken},
		{name: "expired token", header: "Bearer " + expiredToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := auth.NewMiddleware(secret)
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				claims := auth.GetClaims(r.Context())
				if claims == nil || claims.Sub == "" || claims.Role != "operator" {
					t.Error("valid token did not inject the expected claims")
				}
				w.WriteHeader(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodGet, "/shipments/HBX-SEED-0001/updates", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := httptest.NewRecorder()
			middleware.RequireSessionAuth(next).ServeHTTP(w, req)

			if called != tt.wantValid {
				t.Fatalf("next handler called = %t, want %t", called, tt.wantValid)
			}
			if tt.wantValid {
				if w.Code != http.StatusNoContent {
					t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
				}
				return
			}
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, http.StatusUnauthorized, w.Body.String())
			}
			if got := w.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			var body sessionErrorBody
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatalf("decode error body: %v", err)
			}
			if body.Error.Code != "SESSION_EXPIRED" {
				t.Errorf("error.code = %q, want SESSION_EXPIRED", body.Error.Code)
			}
			if body.Error.Message != "Session expired; sign in again." {
				t.Errorf("error.message = %q, want %q", body.Error.Message, "Session expired; sign in again.")
			}
		})
	}
}
