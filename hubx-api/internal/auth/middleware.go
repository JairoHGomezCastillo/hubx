package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// contextKey is used for storing claims in request context.
type contextKey string

const (
	// ClaimsContextKey is the key for storing JWT claims in the request context.
	ClaimsContextKey contextKey = "claims"
)

// Middleware provides authentication middleware.
type Middleware struct {
	jwtSecret string
}

// NewMiddleware creates a new authentication middleware.
func NewMiddleware(jwtSecret string) *Middleware {
	return &Middleware{jwtSecret: jwtSecret}
}

// RequireAuth is middleware that validates JWT tokens on protected endpoints.
// It extracts the Authorization header, validates the JWT, and injects claims
// into the request context. Returns 401 if the token is missing, invalid, or expired.
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			m.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header required")
			return
		}
		
		// Check Bearer scheme
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			m.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization header format")
			return
		}
		
		token := parts[1]
		
		// Validate token
		claims, err := ValidateToken(token, m.jwtSecret)
		if err != nil {
			if err == ErrExpiredToken {
				m.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Session expired. Please sign in again.")
				return
			}
			// Invalid token (bad signature, malformed, etc.)
			m.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token")
			return
		}
		
		// Inject claims into context
		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// writeError writes a JSON error response.
func (m *Middleware) writeError(w http.ResponseWriter, status int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error:   errorCode,
		Message: message,
	})
}

// GetClaims extracts JWT claims from the request context.
// Returns nil if no claims are present (middleware was not applied).
func GetClaims(ctx context.Context) *Claims {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}
