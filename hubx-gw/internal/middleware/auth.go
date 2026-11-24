// Package middleware provides HTTP middleware for hubx-gw.
package middleware

import (
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

// errorResponse represents an error response.
type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
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
			
			// Extract Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header required")
				return
			}
			
			// Check Bearer scheme
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization header format")
				return
			}
			
			token := parts[1]
			
			// Validate token
			if err := validateToken(token, jwtSecret); err != nil {
				if err == ErrExpiredToken {
					writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Session expired. Please sign in again.")
					return
				}
				// Invalid token (bad signature, malformed, etc.)
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token")
				return
			}
			
			// Token is valid; forward request to upstream with Authorization header intact
			next.ServeHTTP(w, r)
		})
	}
}

// validateToken verifies the JWT signature and checks expiration.
// Returns ErrExpiredToken if expired, ErrInvalidToken if invalid.
func validateToken(tokenString, secret string) error {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method is HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	
	if err != nil {
		// Check if the error is due to expiration
		if errors.Is(err, jwt.ErrTokenExpired) {
			return ErrExpiredToken
		}
		return ErrInvalidToken
	}
	
	if !token.Valid {
		return ErrInvalidToken
	}
	
	return nil
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
