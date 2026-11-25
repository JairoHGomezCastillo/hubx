package test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"hubx-gw/internal/middleware"
	"hubx-gw/internal/proxy"
)

var gatewayHeaders = map[string]string{
	"X-Content-Type-Options":    "nosniff",
	"X-Frame-Options":           "DENY",
	"Content-Security-Policy":   "default-src 'none'; frame-ancestors 'none'",
	"Referrer-Policy":           "no-referrer",
	"Strict-Transport-Security": "max-age=31536000",
	"Cache-Control":             "no-store",
}

func TestProxy_GatewayHeadersAreAuthoritative(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "sniff-please")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Content-Security-Policy", "default-src *")
		h.Set("Referrer-Policy", "unsafe-url")
		h.Set("Strict-Transport-Security", "max-age=1; includeSubDomains")
		h.Set("Cache-Control", "public, max-age=600")
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Credentials", "true")
		h.Set("Access-Control-Expose-Headers", "X-Secret")
		h.Set("Access-Control-Max-Age", "86400")
		h.Set("Vary", "Accept-Encoding")
		h.Set("X-Custom", "kept")
		h.Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	chain := middleware.SecurityHeaders(
		middleware.CORS([]string{"http://localhost:5173"})(proxy.New(upstream.URL, "s")))

	req := httptest.NewRequest(http.MethodGet, "/api/anything", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	for name, want := range gatewayHeaders {
		if got := rec.Header().Values(name); len(got) != 1 || got[0] != want {
			t.Errorf("%s = %q, want exactly [%q]", name, got, want)
		}
	}
	if got := rec.Header().Values("Access-Control-Allow-Origin"); len(got) != 1 || got[0] != "http://localhost:5173" {
		t.Errorf("Access-Control-Allow-Origin = %q, want only the gateway's value", got)
	}
	for _, name := range []string{"Access-Control-Allow-Credentials", "Access-Control-Expose-Headers", "Access-Control-Max-Age"} {
		if _, ok := rec.Header()[name]; ok {
			t.Errorf("%s from the API must be removed", name)
		}
	}
	vary := strings.Join(rec.Header().Values("Vary"), ",")
	if !strings.Contains(vary, "Origin") || !strings.Contains(vary, "Accept-Encoding") {
		t.Errorf("Vary = %q, want both Origin (gateway) and Accept-Encoding (API)", vary)
	}
	if rec.Header().Get("X-Custom") != "kept" || rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("unrelated upstream headers must pass through: %v", rec.Header())
	}
	if rec.Body.String() != `{"ok":true}` {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestProxy_ApiAccessControlHeadersAreRemovedForOtherOrigins(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://evil.example")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	chain := middleware.CORS([]string{"http://localhost:5173"})(proxy.New(upstream.URL, "s"))
	req := httptest.NewRequest(http.MethodGet, "/api/anything", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	if got := rec.Header().Values("Access-Control-Allow-Origin"); len(got) != 0 {
		t.Errorf("Access-Control-Allow-Origin = %q, want none: the gateway is authoritative", got)
	}
	if !slices.Contains(rec.Header().Values("Vary"), "Origin") {
		t.Errorf("Vary = %q, must keep Origin", rec.Header().Values("Vary"))
	}
}

func TestProxy_AuthorizationAndForwardedForAreForwardedAsBefore(t *testing.T) {
	var gotAuth, gotXFF, gotSecret string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotXFF = r.Header.Get("X-Forwarded-For")
		gotSecret = r.Header.Get(proxy.GatewaySecretHeader)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	req.RemoteAddr = "192.0.2.7:1234"
	req.Header.Set("Authorization", "Bearer abc.def.ghi")
	proxy.New(upstream.URL, "shared").ServeHTTP(httptest.NewRecorder(), req)

	if gotAuth != "Bearer abc.def.ghi" {
		t.Errorf("Authorization = %q, want untouched", gotAuth)
	}
	if gotXFF != "192.0.2.7" {
		t.Errorf("X-Forwarded-For = %q, want the unchanged ReverseProxy behaviour", gotXFF)
	}
	if gotSecret != "shared" {
		t.Errorf("secret = %q", gotSecret)
	}
}

func TestProxy_BadGatewayStillCarriesGatewayHeaders(t *testing.T) {
	chain := middleware.SecurityHeaders(proxy.New("http://localhost:1", "s"))
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/x", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
	for name, want := range gatewayHeaders {
		if got := rec.Header().Values(name); len(got) != 1 || got[0] != want {
			t.Errorf("%s = %q, want exactly [%q]", name, got, want)
		}
	}
}
