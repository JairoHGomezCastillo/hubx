package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hubx-gw/internal/middleware"
)

var contractSecurityHeaders = map[string]string{
	"X-Content-Type-Options":    "nosniff",
	"X-Frame-Options":           "DENY",
	"Content-Security-Policy":   "default-src 'none'; frame-ancestors 'none'",
	"Referrer-Policy":           "no-referrer",
	"Strict-Transport-Security": "max-age=31536000",
	"Cache-Control":             "no-store",
}

func assertSecurityHeaders(t *testing.T, h http.Header) {
	t.Helper()
	for name, want := range contractSecurityHeaders {
		values := h.Values(name)
		if len(values) != 1 || values[0] != want {
			t.Errorf("%s = %q, want exactly [%q]", name, values, want)
		}
	}
}

func TestSecurityHeaders_AreSetOnEveryStatus(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusBadGateway, http.StatusTooManyRequests, http.StatusUnauthorized} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			h := middleware.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/anything", nil))
			if rec.Code != status {
				t.Fatalf("status = %d, want %d", rec.Code, status)
			}
			assertSecurityHeaders(t, rec.Header())
		})
	}
}

func TestSecurityHeaders_AppearOnceEvenIfDownstreamSetsThemAgain(t *testing.T) {
	h := middleware.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	// A handler that sets the header with Set replaces ours: the proxy is the
	// only downstream writer and removes upstream copies (see proxy tests).
	if got := rec.Header().Values("X-Frame-Options"); len(got) != 1 {
		t.Errorf("X-Frame-Options = %q, want a single value", got)
	}
}

func okHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestCORS_NonPreflightRequests(t *testing.T) {
	allowed := []string{"http://localhost:5173", "https://hubx.example"}
	tests := []struct {
		name       string
		origin     []string
		wantOrigin string
	}{
		{"allowed origin", []string{"http://localhost:5173"}, "http://localhost:5173"},
		{"second allowed origin", []string{"https://hubx.example"}, "https://hubx.example"},
		{"other origin", []string{"https://evil.example"}, ""},
		{"null origin", []string{"null"}, ""},
		{"different case", []string{"HTTP://LOCALHOST:5173"}, ""},
		{"prefix of an allowed origin", []string{"http://localhost:51730"}, ""},
		{"no origin", nil, ""},
		{"two origin headers", []string{"http://localhost:5173", "https://evil.example"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			h := middleware.CORS(allowed)(okHandler(&called))
			req := httptest.NewRequest(http.MethodGet, "/api/shipments", nil)
			for _, o := range tt.origin {
				req.Header.Add("Origin", o)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if !called {
				t.Error("a non-preflight request must be processed normally")
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.wantOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.wantOrigin)
			}
			if vary := rec.Header().Values("Vary"); len(vary) != 1 || vary[0] != "Origin" {
				t.Errorf("Vary = %q, want [Origin] always", vary)
			}
			if _, ok := rec.Header()["Access-Control-Allow-Credentials"]; ok {
				t.Error("Access-Control-Allow-Credentials must never be sent")
			}
		})
	}
}

func TestCORS_PreflightFromAllowedOrigin(t *testing.T) {
	called := false
	h := middleware.CORS([]string{"http://localhost:5173"})(okHandler(&called))
	req := httptest.NewRequest(http.MethodOptions, "/api/shipments", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if called {
		t.Fatal("a preflight must never reach the next handler")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	want := map[string]string{
		"Access-Control-Allow-Origin":  "http://localhost:5173",
		"Access-Control-Allow-Methods": "GET, POST",
		"Access-Control-Allow-Headers": "Authorization, Content-Type",
		"Vary":                         "Origin",
	}
	for name, value := range want {
		if got := rec.Header().Values(name); len(got) != 1 || got[0] != value {
			t.Errorf("%s = %q, want [%q]", name, got, value)
		}
	}
	for _, name := range []string{"Access-Control-Allow-Credentials", "Access-Control-Max-Age"} {
		if _, ok := rec.Header()[name]; ok {
			t.Errorf("%s must not be sent", name)
		}
	}
}

func TestCORS_PreflightFromOtherOriginIsRefused(t *testing.T) {
	origins := []string{"https://evil.example", "null", "http://localhost:5173/"}
	for _, origin := range origins {
		t.Run(origin, func(t *testing.T) {
			called := false
			h := middleware.CORS([]string{"http://localhost:5173"})(okHandler(&called))
			req := httptest.NewRequest(http.MethodOptions, "/api/shipments", nil)
			req.Header.Set("Origin", origin)
			req.Header.Set("Access-Control-Request-Method", "GET")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if called {
				t.Fatal("a refused preflight must not reach the next handler")
			}
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403", rec.Code)
			}
			for name := range rec.Header() {
				if len(name) >= 14 && name[:14] == "Access-Control" {
					t.Errorf("unexpected %s on a refused preflight", name)
				}
			}
			if vary := rec.Header().Values("Vary"); len(vary) != 1 || vary[0] != "Origin" {
				t.Errorf("Vary = %q, want [Origin]", vary)
			}
		})
	}
}

func TestCORS_EmptyAllowListGrantsNothing(t *testing.T) {
	called := false
	h := middleware.CORS(nil)(okHandler(&called))

	pre := httptest.NewRequest(http.MethodOptions, "/api/x", nil)
	pre.Header.Set("Origin", "http://localhost:5173")
	pre.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, pre)
	if rec.Code != http.StatusForbidden || called {
		t.Errorf("preflight status = %d called=%v, want 403 and not called", rec.Code, called)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want none", got)
	}
}

func TestCORS_OptionsThatIsNotAPreflightIsProcessed(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		request string
	}{
		{"no origin", "", "GET"},
		{"origin without request method", "http://localhost:5173", ""},
		{"neither", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			h := middleware.CORS([]string{"http://localhost:5173"})(okHandler(&called))
			req := httptest.NewRequest(http.MethodOptions, "/api/shipments", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.request != "" {
				req.Header.Set("Access-Control-Request-Method", tt.request)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if !called {
				t.Error("an OPTIONS that is not a preflight must reach the next handler (and the JWT)")
			}
		})
	}
}

func TestCORS_PostFromOtherOriginIsProcessedWithoutPermission(t *testing.T) {
	called := false
	h := middleware.CORS([]string{"http://localhost:5173"})(okHandler(&called))
	req := httptest.NewRequest(http.MethodPost, "/api/shipments", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !called || rec.Code != http.StatusOK {
		t.Errorf("called=%v status=%d, want the request processed", called, rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("another origin must get no permission header")
	}
}
