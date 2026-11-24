package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	domain "hubx-api/internal/auth"
)

// RequireSessionAuth validates the JWT exactly as RequireAuth does, but reports
// every rejected session with the nested SESSION_EXPIRED error body required by
// the HU-03 update contract. It is wired only on the HU-03 routes so HU-01 and
// HU-02 keep their flat UNAUTHORIZED responses. The API still validates the JWT
// on its own, independently of the gateway (Constitution II).
func (m *Middleware) RequireSessionAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeSessionError(w)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeSessionError(w)
			return
		}

		claims, err := domain.ValidateToken(parts[1], m.jwtSecret)
		if err != nil {
			writeSessionError(w)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type sessionErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type sessionErrorResponse struct {
	Error sessionErrorDetail `json:"error"`
}

func writeSessionError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(sessionErrorResponse{
		Error: sessionErrorDetail{
			Code:    "SESSION_EXPIRED",
			Message: "Session expired; sign in again.",
		},
	})
}
