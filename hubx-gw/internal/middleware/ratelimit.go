package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hubx-gw/internal/ratelimit"
)

const (
	signInPath = "/api/auth/sign-in"
	// maxSignInBody is the most the gateway reads of a sign-in body.
	maxSignInBody = 8192

	rateLimitedBody  = `{"error":"RATE_LIMITED","message":"Too many requests. Please try again later."}`
	tooLargeBody     = `{"error":"PAYLOAD_TOO_LARGE","message":"Request body too large"}`
	missingFieldsMsg = `{"error":"MISSING_REQUIRED_FIELDS","message":"Email and password are required"}`
)

// signInPayload mirrors the API's sign-in request (hubx-api handlers/auth) so
// that both decode the same email from the same bytes: same decoder, same keys,
// same handling of duplicates, case and trailing data.
type signInPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RateLimit limits requests after the JWT middleware. Requests with a validated
// subject (read only from the request context, never from headers) use users.
// POST /api/auth/sign-in uses signIn, keyed by the SHA-256 of the normalised
// email. Everything else (health, requests that have no identity) is not limited.
// Emails are never logged.
func RateLimit(users, signIn ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if sub, ok := SubjectFromContext(r.Context()); ok {
				if allowed, retry := users.Allow(sub); !allowed {
					writeRateLimited(w, retry)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			if r.Method == http.MethodPost && r.URL.Path == signInPath {
				limitSignIn(signIn, next, w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func limitSignIn(signIn ratelimit.Limiter, next http.Handler, w http.ResponseWriter, r *http.Request) {
	if r.Body == nil || r.Body == http.NoBody {
		next.ServeHTTP(w, r)
		return
	}

	data, err := io.ReadAll(io.LimitReader(r.Body, maxSignInBody+1))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, missingFieldsMsg)
		return
	}
	if len(data) > maxSignInBody {
		writeJSON(w, http.StatusRequestEntityTooLarge, tooLargeBody)
		return
	}

	// Restore the body byte for byte for the API.
	r.Body = io.NopCloser(bytes.NewReader(data))
	r.ContentLength = int64(len(data))
	r.TransferEncoding = nil

	var payload signInPayload
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&payload); err != nil {
		next.ServeHTTP(w, r) // the API answers its own 400; not counted
		return
	}
	email := strings.ToLower(strings.TrimSpace(payload.Email)) // same as NormalizeEmail in the API
	if email == "" {
		next.ServeHTTP(w, r)
		return
	}

	sum := sha256.Sum256([]byte(email))
	if allowed, retry := signIn.Allow(hex.EncodeToString(sum[:])); !allowed {
		writeRateLimited(w, retry)
		return
	}
	next.ServeHTTP(w, r)
}

func writeRateLimited(w http.ResponseWriter, retry time.Duration) {
	w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(retry)))
	writeJSON(w, http.StatusTooManyRequests, rateLimitedBody)
}

// retryAfterSeconds rounds up to whole seconds and clamps to 1..60.
func retryAfterSeconds(d time.Duration) int {
	secs := int((d + time.Second - 1) / time.Second)
	if secs < 1 {
		return 1
	}
	if secs > 60 {
		return 60
	}
	return secs
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}
