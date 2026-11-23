package auth

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

// GatewaySecretHeader is the header the gateway injects to identify itself.
// The API rejects any request that does not carry it with the shared value,
// so legitimate traffic always enters through the gateway (Constitution VI).
const GatewaySecretHeader = "X-Gateway-Secret"

// RequireGatewaySecret returns middleware that verifies the shared gateway
// secret on every request before any route handler runs. The comparison is
// constant-time to avoid leaking the secret through timing. A missing or
// mismatched header is answered with 401 and the request is not forwarded.
func RequireGatewaySecret(gatewaySecret string) func(http.Handler) http.Handler {
	expected := []byte(gatewaySecret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := []byte(r.Header.Get(GatewaySecretHeader))
			if subtle.ConstantTimeCompare(got, expected) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(errorResponse{
					Error:   "UNAUTHORIZED",
					Message: "Request must come through the gateway",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
