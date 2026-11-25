// Package middleware provides HTTP middleware for hubx-gw.
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when the JWT is malformed or signature is invalid.
	ErrInvalidToken = errors.New("invalid token")

	// ErrExpiredToken is returned when the JWT exp claim is in the past.
	ErrExpiredToken = errors.New("expired token")
)

// Claims represents the JWT payload for HUBX sessions (matches API structure).
type Claims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// errorResponse represents the flat error response used by HU-01/HU-02.
type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// sessionErrorResponse represents the nested error body required by HU-03.
type sessionErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// usesSessionErrorContract reports whether the route's HTTP error contract
// requires the nested SESSION_EXPIRED body. It only inspects method and path:
// no business rule is applied here, the API remains authoritative for the
// shipment and ledger state.
func usesSessionErrorContract(r *http.Request) bool {
	if r.Method == http.MethodGet && r.URL.Path == "/api/ground-partners" {
		return true
	}
	if r.Method != http.MethodPost {
		return false
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	return len(parts) == 4 && parts[0] == "api" && parts[1] == "shipments" && parts[2] != "" && parts[3] == "updates"
}

// JWTValidation returns middleware that validates JWT tokens on all routes except
// the excluded paths. It validates the JWT signature and expiration, then forwards
// the request (with the Authorization header intact) to the upstream API.
// The API validates the JWT independently (defense in depth, Constitution II).
func JWTValidation(jwtSecret string, excludedPaths map[string]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path is excluded from JWT validation
			if excludedPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// The HU-03 contract defines a nested 401 body; every other route
			// keeps the flat UNAUTHORIZED body.
			sessionBody := usesSessionErrorContract(r)

			// Extract Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				rejectUnauthorized(w, sessionBody, "Authorization header required")
				return
			}

			// Check Bearer scheme
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				rejectUnauthorized(w, sessionBody, "Invalid authorization header format")
				return
			}

			token := parts[1]

			// Validate token
			sub, err := validateToken(token, jwtSecret)
			if err != nil {
				if err == ErrExpiredToken {
					rejectUnauthorized(w, sessionBody, "Session expired. Please sign in again.")
					return
				}
				// Invalid token (bad signature, malformed, etc.)
				rejectUnauthorized(w, sessionBody, "Invalid token")
				return
			}
			// A valid signature with no subject identifies nobody: it cannot be limited.
			if sub == "" {
				rejectUnauthorized(w, sessionBody, "Invalid token")
				return
			}

			// Token is valid; forward request to upstream with Authorization header intact.
			// The validated subject travels in the context for the rate limiter.
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), subjectKey{}, sub)))
		})
	}
}

// subjectKey is the unexported context key of the validated subject.
type subjectKey struct{}

// SubjectFromContext returns the subject of a token validated by JWTValidation.
// Requests on excluded paths have no subject.
func SubjectFromContext(ctx context.Context) (string, bool) {
	sub, ok := ctx.Value(subjectKey{}).(string)
	return sub, ok
}

// validateToken verifies the JWT signature and checks expiration and returns
// the subject. Returns ErrExpiredToken if expired, ErrInvalidToken if invalid.
func validateToken(tokenString, secret string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method is HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})

	if err != nil {
		// Check if the error is due to expiration
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrExpiredToken
		}
		return "", ErrInvalidToken
	}

	if !token.Valid {
		return "", ErrInvalidToken
	}

	return claims.Sub, nil
}

// rejectUnauthorized writes a 401 in the shape required by the route's contract.
func rejectUnauthorized(w http.ResponseWriter, sessionBody bool, message string) {
	if sessionBody {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		var body sessionErrorResponse
		body.Error.Code = "SESSION_EXPIRED"
		body.Error.Message = "Session expired; sign in again."
		_ = json.NewEncoder(w).Encode(body)
		return
	}
	writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error:   errorCode,
		Message: message,
	})
}
