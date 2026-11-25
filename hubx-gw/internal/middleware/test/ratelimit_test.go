package test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/mock/gomock"

	"hubx-gw/internal/middleware"
	"hubx-gw/internal/ratelimit/mocks"
)

const (
	rlSecret      = "test-secret-key-32-bytes-long!!"
	refusalBody   = `{"error":"RATE_LIMITED","message":"Too many requests. Please try again later."}`
	tooLargeBody  = `{"error":"PAYLOAD_TOO_LARGE","message":"Request body too large"}`
	missingFields = `{"error":"MISSING_REQUIRED_FIELDS","message":"Email and password are required"}`
)

var excluded = map[string]bool{
	"/health":           true,
	"/api/health":       true,
	"/api/auth/sign-in": true,
}

func tokenForSub(t *testing.T, sub string) string {
	t.Helper()
	claims := middleware.Claims{
		Sub:  sub,
		Role: "operator",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(rlSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return s
}

func emailKey(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

type rig struct {
	users  *mocks.MockLimiter
	signIn *mocks.MockLimiter
	chain  http.Handler
	seen   *http.Request
	body   []byte
	calls  int
}

func newRig(t *testing.T) *rig {
	t.Helper()
	ctrl := gomock.NewController(t)
	r := &rig{users: mocks.NewMockLimiter(ctrl), signIn: mocks.NewMockLimiter(ctrl)}
	next := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.calls++
		r.seen = req
		if req.Body != nil {
			r.body, _ = io.ReadAll(req.Body)
		}
		w.WriteHeader(http.StatusOK)
	})
	r.chain = middleware.JWTValidation(rlSecret, excluded)(middleware.RateLimit(r.users, r.signIn)(next))
	return r
}

func (r *rig) do(req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	r.chain.ServeHTTP(rec, req)
	return rec
}

func authed(t *testing.T, method, target, sub string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("Authorization", "Bearer "+tokenForSub(t, sub))
	return req
}

func signInRequest(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/api/auth/sign-in", strings.NewReader(body))
}

func assertRefusal(t *testing.T, rec *httptest.ResponseRecorder, retryAfter string) {
	t.Helper()
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if got := rec.Header().Values("Retry-After"); len(got) != 1 || got[0] != retryAfter {
		t.Errorf("Retry-After = %q, want [%q]", got, retryAfter)
	}
	if got := rec.Header().Values("Content-Type"); len(got) != 1 || got[0] != "application/json" {
		t.Errorf("Content-Type = %q, want exactly [application/json]", got)
	}
	if rec.Body.String() != refusalBody {
		t.Errorf("body = %q, want %q", rec.Body.String(), refusalBody)
	}
}

func TestRateLimit_UserIsKeyedByValidatedSub(t *testing.T) {
	r := newRig(t)
	r.users.EXPECT().Allow("user-1").Return(true, time.Duration(0))

	req := authed(t, http.MethodGet, "/api/shipments", "user-1")
	// Headers that a caller controls must never choose the key.
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	req.Header.Set("X-Real-IP", "203.0.113.10")
	req.Header.Set("X-User-Id", "someone-else")
	req.RemoteAddr = "198.51.100.1:5555"

	rec := r.do(req)
	if rec.Code != http.StatusOK || r.calls != 1 {
		t.Fatalf("status = %d calls = %d, want forwarded", rec.Code, r.calls)
	}
}

func TestRateLimit_RefusalIsNotForwarded(t *testing.T) {
	r := newRig(t)
	r.users.EXPECT().Allow("user-1").Return(false, 1500*time.Millisecond)

	rec := r.do(authed(t, http.MethodGet, "/api/shipments", "user-1"))
	assertRefusal(t, rec, "2")
	if r.calls != 0 {
		t.Error("a refused request must not be forwarded")
	}
}

func TestRateLimit_RetryAfterIsRoundedUpAndClamped(t *testing.T) {
	tests := []struct {
		retry time.Duration
		want  string
	}{
		{0, "1"},
		{time.Millisecond, "1"},
		{time.Second, "1"},
		{time.Second + time.Millisecond, "2"},
		{3 * time.Second, "3"},
		{59*time.Second + 200*time.Millisecond, "60"},
		{time.Minute, "60"},
		{90 * time.Second, "60"},
	}
	for _, tt := range tests {
		t.Run(tt.retry.String(), func(t *testing.T) {
			r := newRig(t)
			r.users.EXPECT().Allow("u").Return(false, tt.retry)
			assertRefusal(t, r.do(authed(t, http.MethodGet, "/api/shipments", "u")), tt.want)
		})
	}
}

func TestRateLimit_NoIdentityNoLimit(t *testing.T) {
	// No mock expectations: any call to a limiter fails the test.
	paths := []struct {
		method, target string
		withToken      bool
	}{
		{http.MethodGet, "/health", false},
		{http.MethodGet, "/health", true},
		{http.MethodGet, "/api/health", false},
		{http.MethodGet, "/api/health", true},
	}
	for _, p := range paths {
		t.Run(p.method+" "+p.target, func(t *testing.T) {
			r := newRig(t)
			var req *http.Request
			if p.withToken {
				req = authed(t, p.method, p.target, "u")
			} else {
				req = httptest.NewRequest(p.method, p.target, nil)
			}
			if rec := r.do(req); rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
		})
	}
}

func TestRateLimit_InvalidSessionConsumesNothing(t *testing.T) {
	forged := httptest.NewRequest(http.MethodGet, "/api/shipments", nil)
	forged.Header.Set("Authorization", "Bearer "+strings.Replace(tokenForSub(t, "victim"), "A", "B", 1)+"x")

	r := newRig(t) // no expectations: the limiter must not be called
	if rec := r.do(forged); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if r.calls != 0 {
		t.Error("forged request reached the next handler")
	}
}

func TestRateLimit_SignInKeyIsHashOfNormalisedEmail(t *testing.T) {
	body := `{"email":"  User@HubX.CO ","password":"s3cret"}`
	r := newRig(t)
	r.signIn.EXPECT().Allow(emailKey("user@hubx.co")).Return(true, time.Duration(0))

	req := signInRequest(body)
	rec := r.do(req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if string(r.body) != body {
		t.Errorf("forwarded body = %q, want it byte for byte %q", r.body, body)
	}
	if r.seen.ContentLength != int64(len(body)) {
		t.Errorf("ContentLength = %d, want %d", r.seen.ContentLength, len(body))
	}
	if len(r.seen.TransferEncoding) != 0 {
		t.Errorf("TransferEncoding = %v, want cleared", r.seen.TransferEncoding)
	}
}

func TestRateLimit_SignInChunkedBodyIsRestoredWithLength(t *testing.T) {
	body := `{"email":"a@b.co","password":"x"}`
	r := newRig(t)
	r.signIn.EXPECT().Allow(emailKey("a@b.co")).Return(true, time.Duration(0))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/sign-in", io.NopCloser(strings.NewReader(body)))
	req.ContentLength = -1
	req.TransferEncoding = []string{"chunked"}
	if rec := r.do(req); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if string(r.body) != body || r.seen.ContentLength != int64(len(body)) || r.seen.TransferEncoding != nil {
		t.Errorf("body=%q length=%d te=%v", r.body, r.seen.ContentLength, r.seen.TransferEncoding)
	}
}

func TestRateLimit_SignInRefusalIsIdenticalAndNotForwarded(t *testing.T) {
	var bodies []string
	for _, email := range []string{"registered@hubx.co", "unknown@hubx.co"} {
		r := newRig(t)
		r.signIn.EXPECT().Allow(emailKey(email)).Return(false, 12*time.Second)
		rec := r.do(signInRequest(`{"email":"` + email + `","password":"x"}`))
		assertRefusal(t, rec, "12")
		if r.calls != 0 {
			t.Error("a refused sign-in must not be forwarded")
		}
		bodies = append(bodies, rec.Body.String())
	}
	if bodies[0] != bodies[1] {
		t.Errorf("refusals differ: %q vs %q", bodies[0], bodies[1])
	}
}

// The gateway must read the same email as the API: same decoder, same struct.
func TestRateLimit_SignInParsesLikeTheAPI(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"duplicate keys, last wins", `{"email":"first@b.co","email":"Last@B.co","password":"x"}`, "last@b.co"},
		{"case-insensitive key", `{"EMAIL":"A@B.co","password":"x"}`, "a@b.co"},
		{"mixed case key", `{"Email":" A@B.co ","Password":"x"}`, "a@b.co"},
		{"trailing data is ignored", `{"email":"a@b.co","password":"x"}{"email":"z@z.co"}`, "a@b.co"},
		{"trailing garbage", `{"email":"a@b.co","password":"x"} trailing`, "a@b.co"},
		{"unicode escape", `{"email":"\u0041@b.co","password":"x"}`, "a@b.co"},
		{"tab and newline padding", "{\"email\":\"\\t a@b.co \\n\",\"password\":\"x\"}", "a@b.co"},
		{"unknown fields", `{"email":"a@b.co","password":"x","extra":[1,2]}`, "a@b.co"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig(t)
			r.signIn.EXPECT().Allow(emailKey(tt.want)).Return(true, time.Duration(0))
			if rec := r.do(signInRequest(tt.body)); rec.Code != http.StatusOK {
				t.Fatalf("status = %d", rec.Code)
			}
			if string(r.body) != tt.body {
				t.Errorf("forwarded body = %q, want %q", r.body, tt.body)
			}
		})
	}
}

func TestRateLimit_SignInWithoutReadableEmailIsForwardedUncounted(t *testing.T) {
	bodies := []string{
		``,
		`not json`,
		`{}`,
		`null`,
		`[1,2]`,
		`{"email":""}`,
		`{"email":"   "}`,
		`{"email":5}`,
		`{"email":"a@b.co","password":5}`,
		`{"email":"a@b.co"`,
	}
	for _, body := range bodies {
		t.Run(body, func(t *testing.T) {
			r := newRig(t) // no expectation: no limiter call
			rec := r.do(signInRequest(body))
			if rec.Code != http.StatusOK || r.calls != 1 {
				t.Fatalf("status=%d calls=%d, want forwarded for the API's own 400", rec.Code, r.calls)
			}
			if string(r.body) != body {
				t.Errorf("forwarded body = %q, want %q", r.body, body)
			}
		})
	}
}

func TestRateLimit_SignInNilBodyIsForwardedUncounted(t *testing.T) {
	r := newRig(t)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sign-in", nil)
	if rec := r.do(req); rec.Code != http.StatusOK || r.calls != 1 {
		t.Fatalf("status=%d calls=%d", rec.Code, r.calls)
	}
}

func paddedBody(total int) string {
	core := `{"email":"a@b.co","password":"x"}`
	return core + strings.Repeat(" ", total-len(core))
}

func TestRateLimit_SignInBodyBound(t *testing.T) {
	t.Run("exactly 8192 bytes is accepted", func(t *testing.T) {
		r := newRig(t)
		r.signIn.EXPECT().Allow(emailKey("a@b.co")).Return(true, time.Duration(0))
		body := paddedBody(8192)
		if rec := r.do(signInRequest(body)); rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		if string(r.body) != body {
			t.Error("body altered")
		}
	})

	t.Run("8193 bytes is refused with 413", func(t *testing.T) {
		r := newRig(t) // no limiter call, no forwarding
		rec := r.do(signInRequest(paddedBody(8193)))
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413", rec.Code)
		}
		if rec.Body.String() != tooLargeBody {
			t.Errorf("body = %q", rec.Body.String())
		}
		if got := rec.Header().Values("Content-Type"); len(got) != 1 || got[0] != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		if r.calls != 0 {
			t.Error("an oversized body must not be forwarded")
		}
	})

	t.Run("huge declared length reads at most the bound", func(t *testing.T) {
		r := newRig(t)
		counter := &countingReader{}
		req := httptest.NewRequest(http.MethodPost, "/api/auth/sign-in", counter)
		req.ContentLength = 1 << 30
		rec := r.do(req)
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413", rec.Code)
		}
		if counter.n > 8193+4096 {
			t.Errorf("read %d bytes, want about 8193", counter.n)
		}
	})
}

type countingReader struct{ n int }

func (c *countingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'a'
	}
	c.n += len(p)
	return len(p), nil
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

func TestRateLimit_SignInBodyReadErrorIs400(t *testing.T) {
	r := newRig(t)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sign-in", failingReader{})
	rec := r.do(req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if rec.Body.String() != missingFields {
		t.Errorf("body = %q, want %q", rec.Body.String(), missingFields)
	}
	if r.calls != 0 {
		t.Error("an unreadable body must not be forwarded")
	}
}

func TestRateLimit_OnlyExactPostSignInIsCountedAsSignIn(t *testing.T) {
	body := `{"email":"a@b.co","password":"x"}`
	tests := []struct {
		name, method, target string
	}{
		{"GET", http.MethodGet, "/api/auth/sign-in"},
		{"PUT", http.MethodPut, "/api/auth/sign-in"},
		{"HEAD", http.MethodHead, "/api/auth/sign-in"},
		{"DELETE", http.MethodDelete, "/api/auth/sign-in"},
		{"trailing slash", http.MethodPost, "/api/auth/sign-in/"},
		{"other case", http.MethodPost, "/api/auth/Sign-In"},
		{"double slash", http.MethodPost, "/api//auth/sign-in"},
		{"leading double slash", http.MethodPost, "//api/auth/sign-in"},
		{"dot segment", http.MethodPost, "/api/auth/./sign-in"},
		{"parent segment", http.MethodPost, "/api/auth/x/../sign-in"},
		{"another auth route", http.MethodPost, "/api/auth/sign-up"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Only the RateLimit middleware, with no JWT in front: no limiter
			// may be called, whatever the response is.
			ctrl := gomock.NewController(t)
			u, s := mocks.NewMockLimiter(ctrl), mocks.NewMockLimiter(ctrl)
			h := middleware.RateLimit(u, s)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(tt.method, tt.target, bytes.NewBufferString(body))
			h.ServeHTTP(httptest.NewRecorder(), req)
		})
	}
}

func TestRateLimit_EncodedSignInPathIsStillCounted(t *testing.T) {
	// %2D decodes to '-': Go's URL.Path is the decoded form, which is also what
	// the API's ServeMux matches, so the gateway must count it.
	r := newRig(t)
	r.signIn.EXPECT().Allow(emailKey("a@b.co")).Return(true, time.Duration(0))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/sign%2Din", strings.NewReader(`{"email":"a@b.co","password":"x"}`))
	if rec := r.do(req); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestRateLimit_SignInDoesNotUseUserLimiterEvenWithAToken(t *testing.T) {
	// A token on an excluded path is not validated, so it gives no identity.
	r := newRig(t)
	r.signIn.EXPECT().Allow(emailKey("a@b.co")).Return(true, time.Duration(0))
	req := signInRequest(`{"email":"a@b.co","password":"x"}`)
	req.Header.Set("Authorization", "Bearer "+tokenForSub(t, "someone"))
	if rec := r.do(req); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestRateLimit_RefusalBodyIsValidJSON(t *testing.T) {
	r := newRig(t)
	r.users.EXPECT().Allow("u").Return(false, time.Second)
	rec := r.do(authed(t, http.MethodGet, "/api/shipments", "u"))
	var v map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if v["error"] != "RATE_LIMITED" {
		t.Errorf("error = %q", v["error"])
	}
}
