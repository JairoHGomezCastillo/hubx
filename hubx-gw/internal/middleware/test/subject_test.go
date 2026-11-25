package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"hubx-gw/internal/middleware"
)

func TestJWTValidation_PutsValidatedSubInContext(t *testing.T) {
	var gotSub string
	var gotOK bool
	h := middleware.JWTValidation(rlSecret, excluded)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSub, gotOK = middleware.SubjectFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authed(t, http.MethodGet, "/api/shipments", "user-42"))

	if rec.Code != http.StatusOK || !gotOK || gotSub != "user-42" {
		t.Errorf("status=%d sub=%q ok=%v, want 200 user-42 true", rec.Code, gotSub, gotOK)
	}
}

func TestJWTValidation_ExcludedPathsHaveNoSubject(t *testing.T) {
	for _, path := range []string{"/health", "/api/health", "/api/auth/sign-in"} {
		t.Run(path, func(t *testing.T) {
			var gotOK bool
			h := middleware.JWTValidation(rlSecret, excluded)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, gotOK = middleware.SubjectFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))
			// Even with a valid token: excluded paths are not validated.
			h.ServeHTTP(httptest.NewRecorder(), authed(t, http.MethodGet, path, "user-42"))
			if gotOK {
				t.Error("excluded paths must not carry a subject")
			}
		})
	}
}

func TestJWTValidation_EmptySubIsRejected(t *testing.T) {
	claims := middleware.Claims{RegisteredClaims: jwt.RegisteredClaims{}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(rlSecret))
	if err != nil {
		t.Fatal(err)
	}
	h := middleware.JWTValidation(rlSecret, excluded)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("a token without sub must not reach the next handler")
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/shipments", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	var body errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "UNAUTHORIZED" || body.Message != "Invalid token" {
		t.Errorf("body = %+v, want UNAUTHORIZED / Invalid token", body)
	}
}

func TestSubjectFromContext_AbsentByDefault(t *testing.T) {
	if _, ok := middleware.SubjectFromContext(httptest.NewRequest(http.MethodGet, "/", nil).Context()); ok {
		t.Error("a plain context has no subject")
	}
}
