// Package server builds the gateway's HTTP handler chain. It lives apart from
// cmd/gw so the whole chain and its order can be proven by tests.
package server

import (
	"encoding/json"
	"net/http"
	"time"

	"hubx-gw/internal/config"
	"hubx-gw/internal/middleware"
	"hubx-gw/internal/proxy"
	"hubx-gw/internal/ratelimit"
)

const (
	// window is the sliding window of both limiters.
	window = time.Minute
	// maxKeys bounds the identities remembered by each limiter.
	maxKeys = 10000
)

// New returns the gateway handler. The order is part of the contract:
//
//	SecurityHeaders -> CORS -> JWTValidation -> RateLimit -> mux
//
// so every answer carries the security headers, a preflight needs no session,
// an invalid session consumes no limit and only validated identities are counted.
// now is the injected clock of the limiters. Limit state is in memory, per instance.
func New(cfg config.Config, now func() time.Time) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "hubx-gw",
		})
	})
	proxy.Register(mux, cfg.APIURL, cfg.GatewaySecret)

	excludedPaths := map[string]bool{
		"/health":           true, // Gateway health check
		"/api/health":       true, // API health check (proxied)
		"/api/auth/sign-in": true, // Sign-in endpoint
	}

	// Two independent limiters: a flood of sign-in emails cannot evict signed-in users.
	users := ratelimit.New(cfg.RequestsPerMinute, window, maxKeys, now)
	signIn := ratelimit.New(cfg.SignInRequestsPerMinute, window, maxKeys, now)

	var h http.Handler = mux
	h = middleware.RateLimit(users, signIn)(h)
	h = middleware.JWTValidation(cfg.JWTSecret, excludedPaths)(h)
	h = middleware.CORS(cfg.AllowedOrigins)(h)
	h = middleware.SecurityHeaders(h)
	return h
}
