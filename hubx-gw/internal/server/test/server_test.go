package test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"hubx-gw/internal/config"
	"hubx-gw/internal/middleware"
	"hubx-gw/internal/server"
)

const (
	jwtSecret     = "jwt-secret-for-the-wiring-tests-01"
	gatewaySecret = "gateway-secret-for-the-wiring-tests"
	refusalBody   = `{"error":"RATE_LIMITED","message":"Too many requests. Please try again later."}`
)

var securityHeaders = map[string]string{
	"X-Content-Type-Options":    "nosniff",
	"X-Frame-Options":           "DENY",
	"Content-Security-Policy":   "default-src 'none'; frame-ancestors 'none'",
	"Referrer-Policy":           "no-referrer",
	"Strict-Transport-Security": "max-age=31536000",
	"Cache-Control":             "no-store",
}

type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// upstreamLog records what the fake API received.
type upstreamLog struct {
	mu    sync.Mutex
	paths []string
	count atomic.Int64
}

func (u *upstreamLog) add(p string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.paths = append(u.paths, p)
	u.count.Add(1)
}

func (u *upstreamLog) seen(p string) int {
	u.mu.Lock()
	defer u.mu.Unlock()
	n := 0
	for _, got := range u.paths {
		if got == p {
			n++
		}
	}
	return n
}

type env struct {
	handler  http.Handler
	clock    *clock
	upstream *upstreamLog
}

// newEnv wires the real chain from the real hubx-gw/application.properties,
// pointing api.url to a fake upstream. extra are environment overrides.
func newEnv(t *testing.T, extra map[string]string) *env {
	t.Helper()
	log := &upstreamLog{}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.add(r.URL.Path)
		// The API sets its own, conflicting, headers on purpose.
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(api.Close)

	values := map[string]string{
		"JWT_SECRET":     jwtSecret,
		"GATEWAY_SECRET": gatewaySecret,
		"API_URL":        api.URL,
	}
	for k, v := range extra {
		values[k] = v
	}
	cfg, err := config.Load(filepath.Join("..", "..", ".."), func(k string) string { return values[k] })
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	clk := &clock{now: time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)}
	return &env{handler: server.New(cfg, clk.Now), clock: clk, upstream: log}
}

func token(t *testing.T, sub string) string {
	t.Helper()
	claims := middleware.Claims{
		Sub:              sub,
		Role:             "operator",
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func (e *env) get(path, bearer string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

func (e *env) signIn(body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sign-in", strings.NewReader(body))
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

func assertSecurityHeaders(t *testing.T, label string, h http.Header) {
	t.Helper()
	for name, want := range securityHeaders {
		if got := h.Values(name); len(got) != 1 || got[0] != want {
			t.Errorf("%s: %s = %q, want exactly [%q]", label, name, got, want)
		}
	}
}

func TestChain_RealPropertiesFileRequest121Gets429AndRecoversAfterOneMinute(t *testing.T) {
	e := newEnv(t, nil)
	tok := token(t, "user-1")

	for i := 1; i <= 120; i++ {
		if rec := e.get("/api/shipments", tok, nil); rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i, rec.Code)
		}
	}
	rec := e.get("/api/shipments", tok, nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("request 121: status = %d, want 429", rec.Code)
	}
	if rec.Body.String() != refusalBody {
		t.Errorf("body = %q", rec.Body.String())
	}
	if got := rec.Header().Get("Retry-After"); got != "60" {
		t.Errorf("Retry-After = %q, want 60", got)
	}
	if n := e.upstream.count.Load(); n != 120 {
		t.Errorf("upstream received %d requests, want 120: the refused one must not be forwarded", n)
	}

	// Another user is unaffected.
	if rec := e.get("/api/shipments", token(t, "user-2"), nil); rec.Code != http.StatusOK {
		t.Errorf("another user: status = %d, want 200", rec.Code)
	}

	e.clock.Advance(time.Minute)
	if rec := e.get("/api/shipments", tok, nil); rec.Code != http.StatusOK {
		t.Errorf("after one minute: status = %d, want 200", rec.Code)
	}
}

func TestChain_SpoofedAddressHeadersDoNotAvoidTheLimit(t *testing.T) {
	e := newEnv(t, map[string]string{"RATELIMIT_REQUESTS_PER_MINUTE": "3"})
	tok := token(t, "user-1")
	for i := 0; i < 3; i++ {
		e.get("/api/x", tok, map[string]string{"X-Forwarded-For": fmt.Sprintf("10.0.0.%d", i)})
	}
	rec := e.get("/api/x", tok, map[string]string{"X-Forwarded-For": "203.0.113.77", "X-Real-IP": "203.0.113.78"})
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", rec.Code)
	}
}

func TestChain_ForgedSessionNeverConsumesTheUsersLimit(t *testing.T) {
	e := newEnv(t, map[string]string{"RATELIMIT_REQUESTS_PER_MINUTE": "2"})
	good := token(t, "victim")
	forged := good[:len(good)-3] + "AAA" // broken signature, same claims
	for i := 0; i < 10; i++ {
		rec := e.get("/api/x", forged, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("forged request %d: status = %d, want 401", i, rec.Code)
		}
		assertSecurityHeaders(t, "401", rec.Header())
	}
	if rec := e.get("/api/x", good, nil); rec.Code != http.StatusOK {
		t.Errorf("the user's own request: status = %d, want 200", rec.Code)
	}
	if rec := e.get("/api/x", good, nil); rec.Code != http.StatusOK {
		t.Errorf("the user's second request: status = %d, want 200", rec.Code)
	}
}

func TestChain_HealthIsNeverLimited(t *testing.T) {
	e := newEnv(t, map[string]string{"RATELIMIT_REQUESTS_PER_MINUTE": "1", "RATELIMIT_SIGN_IN_REQUESTS_PER_MINUTE": "1"})
	for _, path := range []string{"/health", "/api/health"} {
		for i := 0; i < 50; i++ {
			if rec := e.get(path, "", nil); rec.Code != http.StatusOK {
				t.Fatalf("%s request %d: status = %d, want 200", path, i+1, rec.Code)
			}
		}
		// Even carrying a valid token: health has no identity.
		for i := 0; i < 5; i++ {
			if rec := e.get(path, token(t, "u"), nil); rec.Code != http.StatusOK {
				t.Fatalf("%s with token: status = %d", path, rec.Code)
			}
		}
	}
}

func TestChain_SignInLimitedPerNormalisedEmail(t *testing.T) {
	e := newEnv(t, map[string]string{"RATELIMIT_SIGN_IN_REQUESTS_PER_MINUTE": "3"})
	for i := 0; i < 3; i++ {
		if rec := e.signIn(`{"email":"user@hubx.co","password":"x"}`); rec.Code != http.StatusOK {
			t.Fatalf("attempt %d: status = %d", i+1, rec.Code)
		}
	}
	rec := e.signIn(`{"email":"  USER@HUBX.CO ","password":"y"}`)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if rec.Body.String() != refusalBody {
		t.Errorf("body = %q", rec.Body.String())
	}
	if got := e.upstream.seen("/auth/sign-in"); got != 3 {
		t.Errorf("upstream saw %d sign-ins, want 3", got)
	}
	// Another email is unaffected, and so are signed-in users.
	if rec := e.signIn(`{"email":"other@hubx.co","password":"x"}`); rec.Code != http.StatusOK {
		t.Errorf("another email: status = %d", rec.Code)
	}
	if rec := e.get("/api/x", token(t, "u"), nil); rec.Code != http.StatusOK {
		t.Errorf("signed-in user while sign-in is exhausted: status = %d", rec.Code)
	}
}

func TestChain_SignInFloodCannotEvictSignedInUsers(t *testing.T) {
	e := newEnv(t, map[string]string{"RATELIMIT_REQUESTS_PER_MINUTE": "1"})
	tok := token(t, "user-1")
	e.get("/api/x", tok, nil)
	for i := 0; i < 200; i++ {
		e.signIn(fmt.Sprintf(`{"email":"flood-%d@hubx.co","password":"x"}`, i))
	}
	if rec := e.get("/api/x", tok, nil); rec.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429: users and sign-in use independent limiters", rec.Code)
	}
}

func TestChain_SignInUnreadableEmailIsNotCounted(t *testing.T) {
	e := newEnv(t, map[string]string{"RATELIMIT_SIGN_IN_REQUESTS_PER_MINUTE": "1"})
	for i := 0; i < 5; i++ {
		if rec := e.signIn(`{"password":"x"}`); rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want forwarded", rec.Code)
		}
	}
	if got := e.upstream.seen("/auth/sign-in"); got != 5 {
		t.Errorf("upstream saw %d, want 5", got)
	}
}

func TestChain_SignInPathVariantsCannotBypassTheCount(t *testing.T) {
	e := newEnv(t, map[string]string{"RATELIMIT_SIGN_IN_REQUESTS_PER_MINUTE": "1"})
	body := `{"email":"user@hubx.co","password":"x"}`
	if rec := e.signIn(body); rec.Code != http.StatusOK {
		t.Fatalf("first attempt: status = %d", rec.Code)
	}
	before := e.upstream.count.Load()

	// Variants that would reach the API's sign-in handler are counted: refused.
	counted := httptest.NewRequest(http.MethodPost, "/api/auth/sign%2Din", strings.NewReader(body))
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, counted)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("encoded path: status = %d, want 429", rec.Code)
	}

	// Variants the API would not route to sign-in need a session: never forwarded.
	for _, target := range []string{
		"/api/auth/sign-in/",
		"/api//auth/sign-in",
		"//api/auth/sign-in",
		"/api/auth/./sign-in",
		"/api/auth/x/../sign-in",
		"/api/auth/Sign-In",
		"/api/auth/sign-in%2F",
		"/API/auth/sign-in",
	} {
		req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
		rec := httptest.NewRecorder()
		e.handler.ServeHTTP(rec, req)
		if rec.Code == http.StatusOK {
			t.Errorf("%s: status = 200, the variant must not reach the API without a session", target)
		}
		assertSecurityHeaders(t, target, rec.Header())
	}
	if after := e.upstream.count.Load(); after != before {
		t.Errorf("upstream received %d extra requests through path variants", after-before)
	}
}

func TestChain_SignInMethodsOtherThanPostAreNotCounted(t *testing.T) {
	e := newEnv(t, map[string]string{"RATELIMIT_SIGN_IN_REQUESTS_PER_MINUTE": "1"})
	body := `{"email":"user@hubx.co","password":"x"}`
	for _, m := range []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		req := httptest.NewRequest(m, "/api/auth/sign-in", strings.NewReader(body))
		e.handler.ServeHTTP(httptest.NewRecorder(), req)
	}
	// None of those consumed the POST allowance.
	if rec := e.signIn(body); rec.Code != http.StatusOK {
		t.Errorf("POST after other methods: status = %d, want 200", rec.Code)
	}
}

func TestChain_OversizedSignInBodyIs413WithHeaders(t *testing.T) {
	e := newEnv(t, nil)
	rec := e.signIn(`{"email":"a@b.co","password":"` + strings.Repeat("x", 9000) + `"}`)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}
	assertSecurityHeaders(t, "413", rec.Header())
	if e.upstream.count.Load() != 0 {
		t.Error("an oversized body must not be forwarded")
	}
}

func TestChain_PreflightNeverReachesJWTLimiterOrAPI(t *testing.T) {
	e := newEnv(t, map[string]string{"RATELIMIT_REQUESTS_PER_MINUTE": "1", "RATELIMIT_SIGN_IN_REQUESTS_PER_MINUTE": "1"})
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodOptions, "/api/shipments", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Access-Control-Request-Method", "GET")
		rec := httptest.NewRecorder()
		e.handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("preflight %d: status = %d, want 204 without a session", i+1, rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
			t.Errorf("Access-Control-Allow-Origin = %q", got)
		}
		assertSecurityHeaders(t, "preflight", rec.Header())
	}
	if e.upstream.count.Load() != 0 {
		t.Error("a preflight must never be forwarded")
	}
	// The limit of a user was not touched by the preflights.
	if rec := e.get("/api/x", token(t, "u"), nil); rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	// Preflight from another origin: refused, and still no session needed.
	req := httptest.NewRequest(http.MethodOptions, "/api/shipments", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
	assertSecurityHeaders(t, "refused preflight", rec.Header())
}

func TestChain_OptionsWithoutPreflightHeadersStillNeedsASession(t *testing.T) {
	e := newEnv(t, nil)
	req := httptest.NewRequest(http.MethodOptions, "/api/shipments", nil)
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestChain_EveryAnswerCarriesTheHeadersOnce(t *testing.T) {
	good := token(t, "user-1")
	tests := []struct {
		name string
		do   func(e *env) *httptest.ResponseRecorder
		want int
	}{
		{"proxied success with conflicting API headers", func(e *env) *httptest.ResponseRecorder { return e.get("/api/shipments", good, nil) }, 200},
		{"gateway health", func(e *env) *httptest.ResponseRecorder { return e.get("/health", "", nil) }, 200},
		{"api health", func(e *env) *httptest.ResponseRecorder { return e.get("/api/health", "", nil) }, 200},
		{"no session", func(e *env) *httptest.ResponseRecorder { return e.get("/api/shipments", "", nil) }, 401},
		{"unknown path with session", func(e *env) *httptest.ResponseRecorder { return e.get("/nope", good, nil) }, 404},
		{"unknown path without session", func(e *env) *httptest.ResponseRecorder { return e.get("/nope", "", nil) }, 401},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t, nil)
			rec := tt.do(e)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
			assertSecurityHeaders(t, tt.name, rec.Header())
			if got := rec.Header().Values("Access-Control-Allow-Origin"); len(got) != 0 {
				t.Errorf("Access-Control-Allow-Origin = %q, want none without an Origin", got)
			}
			if got := rec.Header().Values("Vary"); len(got) == 0 || got[0] != "Origin" {
				t.Errorf("Vary = %q, must start with Origin", got)
			}
		})
	}
}

func TestChain_BadGatewayAnd429CarryTheHeaders(t *testing.T) {
	e := newEnv(t, map[string]string{"API_URL": "http://127.0.0.1:1", "RATELIMIT_REQUESTS_PER_MINUTE": "1"})
	tok := token(t, "u")
	rec := e.get("/api/x", tok, nil)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
	assertSecurityHeaders(t, "502", rec.Header())
	rec = e.get("/api/x", tok, nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	assertSecurityHeaders(t, "429", rec.Header())
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("no Origin sent, no permission expected")
	}
}

func TestChain_CORSOnAnswersForAllowedAndOtherOrigins(t *testing.T) {
	e := newEnv(t, map[string]string{"RATELIMIT_REQUESTS_PER_MINUTE": "1"})
	tok := token(t, "u")

	rec := e.get("/api/x", tok, map[string]string{"Origin": "http://localhost:5173"})
	if got := rec.Header().Values("Access-Control-Allow-Origin"); len(got) != 1 || got[0] != "http://localhost:5173" {
		t.Errorf("allowed origin: Access-Control-Allow-Origin = %q (the API's * must be removed)", got)
	}
	if _, ok := rec.Header()["Access-Control-Allow-Credentials"]; ok {
		t.Error("credentials must never be allowed")
	}
	// The 429 is readable by the approved origin too.
	rec = e.get("/api/x", tok, map[string]string{"Origin": "http://localhost:5173"})
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Errorf("429 for an approved origin: status=%d acao=%q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}

	rec = e.get("/api/y", token(t, "other"), map[string]string{"Origin": "https://evil.example"})
	if got := rec.Header().Values("Access-Control-Allow-Origin"); len(got) != 0 {
		t.Errorf("other origin: Access-Control-Allow-Origin = %q, want none", got)
	}
}

func TestChain_ConcurrentRequestsNeverExceedTheLimit(t *testing.T) {
	e := newEnv(t, map[string]string{"RATELIMIT_REQUESTS_PER_MINUTE": "40"})
	tok := token(t, "shared")

	var ok, refused atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 300; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch e.get("/api/x", tok, nil).Code {
			case http.StatusOK:
				ok.Add(1)
			case http.StatusTooManyRequests:
				refused.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 40 || refused.Load() != 260 {
		t.Errorf("ok=%d refused=%d, want exactly 40 and 260", ok.Load(), refused.Load())
	}
	if n := e.upstream.count.Load(); n != 40 {
		t.Errorf("upstream received %d, want 40", n)
	}
}

func TestChain_AuthorizationAndSecretReachTheAPI(t *testing.T) {
	var gotAuth, gotSecret string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotSecret = r.Header.Get("X-Gateway-Secret")
		w.WriteHeader(http.StatusOK)
	}))
	defer api.Close()
	cfg, err := config.Load(filepath.Join("..", "..", ".."), func(k string) string {
		return map[string]string{"JWT_SECRET": jwtSecret, "GATEWAY_SECRET": gatewaySecret, "API_URL": api.URL}[k]
	})
	if err != nil {
		t.Fatal(err)
	}
	h := server.New(cfg, time.Now)
	tok := token(t, "u")
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	h.ServeHTTP(httptest.NewRecorder(), req)
	if gotAuth != "Bearer "+tok || gotSecret != gatewaySecret {
		t.Errorf("Authorization/secret not forwarded as before: %q %q", gotAuth, gotSecret)
	}
}
