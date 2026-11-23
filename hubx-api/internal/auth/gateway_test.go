package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireGatewaySecret(t *testing.T) {
	const secret = "shared-gateway-secret"

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("reached"))
	})
	handler := RequireGatewaySecret(secret)(next)

	t.Run("accepts request with correct secret", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set(GatewaySecretHeader, secret)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if rec.Body.String() != "reached" {
			t.Errorf("body = %q, want %q", rec.Body.String(), "reached")
		}
	})

	t.Run("rejects request without the header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
		if rec.Body.String() == "reached" {
			t.Error("handler was reached without the gateway secret")
		}
	})

	t.Run("rejects request with wrong secret", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set(GatewaySecretHeader, "wrong-secret")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}
