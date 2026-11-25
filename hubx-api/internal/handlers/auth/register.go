package auth

import (
	"database/sql"
	"net/http"

	domain "hubx-api/internal/auth"
)

// Config holds the configuration values needed by the auth module.
type Config struct {
	JWTSecret       string
	ExpirationHours int
}

// Register wires the auth module into the HTTP mux. It returns the
// authentication middleware for use by other modules.
func Register(mux *http.ServeMux, db *sql.DB, cfg Config) *Middleware {
	// Create repository
	repo := domain.NewPostgresRepository(db)

	// Create service
	service := domain.NewAuthService(repo, cfg.JWTSecret, cfg.ExpirationHours)

	// Create handler
	handler := NewHandler(service)

	// Register routes
	mux.HandleFunc("POST /auth/sign-in", handler.SignIn)

	// Create and return middleware for use by other modules
	middleware := NewMiddleware(cfg.JWTSecret)
	return middleware
}
