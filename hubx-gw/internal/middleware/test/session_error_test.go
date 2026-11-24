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

type sessionErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// expiredTestToken signs a JWT whose exp claim is in the past.
func expiredTestToken(t *testing.T, secret string) string {
	t.Helper()
	now := time.Now()
	claims := middleware.Claims{
		Sub:   "550e8400-e29b-41d4-a716-446655440000",
		Email: "operator@hubx.co",
		Role:  "operator",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Hour)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}
	return signed
}

// TestHU03_GatewaySessionErrorBody verifies that only the HU-03 routes get the
// nested SESSION_EXPIRED body from the gateway; every other route keeps the
// flat UNAUTHORIZED body expected by HU-01/HU-02.
func TestHU03_GatewaySessionErrorBody(t *testing.T) {
	const secret = "test-secret-key-32-bytes-long!!"
	expired := expiredTestToken(t, secret)
	wrongSecretToken, err := generateTestToken("wrong-secret-key-32-bytes-long!!", 1)
	if err != nil {
		t.Fatalf("generate wrong-secret token: %v", err)
	}

	tests := []struct {
		name       string
		method     string
		path       string
		header     string
		wantNested bool
	}{
		{name: "expired token on POST updates", method: http.MethodPost, path: "/api/shipments/HBX-SEED-0001/updates", header: "Bearer " + expired, wantNested: true},
		{name: "expired token on GET ground partners", method: http.MethodGet, path: "/api/ground-partners", header: "Bearer " + expired, wantNested: true},
		{name: "invalid signature on POST updates", method: http.MethodPost, path: "/api/shipments/HBX-SEED-0001/updates", header: "Bearer " + wrongSecretToken, wantNested: true},
		{name: "malformed token on POST updates", method: http.MethodPost, path: "/api/shipments/HBX-SEED-0001/updates", header: "Bearer not-a-valid-jwt", wantNested: true},
		{name: "missing header on POST updates", method: http.MethodPost, path: "/api/shipments/HBX-SEED-0001/updates", wantNested: true},
		{name: "expired token on HU-01 shipment route", method: http.MethodGet, path: "/api/shipments/HBX-SEED-0001", header: "Bearer " + expired},
		{name: "invalid token on HU-02 create route", method: http.MethodPost, path: "/api/shipments", header: "Bearer not-a-valid-jwt"},
	}

	excludedPaths := map[string]bool{
		"/health":           true,
		"/api/health":       true,
		"/api/auth/sign-in": true,
	}
	middlewareFunc := middleware.JWTValidation(secret, excludedPaths)
	handler := middlewareFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)

			if res.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d (body %s)", res.Code, http.StatusUnauthorized, res.Body.String())
			}
			if tt.wantNested {
				var body sessionErrorBody
				if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
					t.Fatalf("decode nested error: %v", err)
				}
				if body.Error.Code != "SESSION_EXPIRED" {
					t.Errorf("error.code = %q, want SESSION_EXPIRED", body.Error.Code)
				}
				if body.Error.Message != "Session expired; sign in again." {
					t.Errorf("error.message = %q, want %q", body.Error.Message, "Session expired; sign in again.")
				}
				return
			}
			var body errorResponse
			if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
				t.Fatalf("decode flat error: %v", err)
			}
			if body.Error != "UNAUTHORIZED" {
				t.Errorf("error = %q, want UNAUTHORIZED", body.Error)
			}
		})
	}
}

// TestHU03_GatewayValidTokenForwarded verifies JWT validation still runs on
// HU-03 routes and leaves Authorization untouched for the API (Constitution II).
func TestHU03_GatewayValidTokenForwarded(t *testing.T) {
	const secret = "test-secret-key-32-bytes-long!!"
	token, err := generateTestToken(secret, 1)
	if err != nil {
		t.Fatalf("generate valid token: %v", err)
	}

	handler := middleware.JWTValidation(secret, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+token {
			t.Errorf("forwarded Authorization = %q, want %q", got, "Bearer "+token)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/shipments/HBX-SEED-0001/updates", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", res.Code, http.StatusNoContent)
	}
}

// TestHU03_GatewayExcludedPathsStayOpen verifies health and sign-in never run
// JWT validation, even for HU-03-shaped paths.
func TestHU03_GatewayExcludedPathsStayOpen(t *testing.T) {
	excludedPaths := map[string]bool{
		"/health":           true,
		"/api/health":       true,
		"/api/auth/sign-in": true,
	}
	handler := middleware.JWTValidation("test-secret-key-32-bytes-long!!", excludedPaths)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, path := range []string{"/health", "/api/health", "/api/auth/sign-in"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Errorf("status = %d, want %d without Authorization", res.Code, http.StatusOK)
			}
		})
	}
}
