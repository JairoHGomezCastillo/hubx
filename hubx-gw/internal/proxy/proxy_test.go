package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProxy(t *testing.T) {
	t.Run("reenvía /api/health al upstream sin el prefijo /api", func(t *testing.T) {
		var gotPath string
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))
		defer upstream.Close()

		handler := New(upstream.URL, "test-secret")

		req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if gotPath != "/health" {
			t.Errorf("upstream recibió path = %q, want %q", gotPath, "/health")
		}
	})

	t.Run("inyecta el secreto compartido hacia el upstream", func(t *testing.T) {
		var gotSecret string
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotSecret = r.Header.Get(GatewaySecretHeader)
			w.WriteHeader(http.StatusOK)
		}))
		defer upstream.Close()

		handler := New(upstream.URL, "s3cr3t-value")

		// El cliente intenta suplantar el secreto; el gateway debe sobrescribirlo.
		req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		req.Header.Set(GatewaySecretHeader, "forged-by-client")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if gotSecret != "s3cr3t-value" {
			t.Errorf("upstream recibió %s = %q, want %q", GatewaySecretHeader, gotSecret, "s3cr3t-value")
		}
	})

	t.Run("responde 502 si el upstream está caído", func(t *testing.T) {
		// Puerto sin servicio escuchando: el proxy no podrá conectarse.
		handler := New("http://localhost:1", "test-secret")

		req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadGateway {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
		}
		if !strings.Contains(rec.Body.String(), "Bad Gateway") {
			t.Errorf("body = %q, want que contenga %q", rec.Body.String(), "Bad Gateway")
		}
	})
}
