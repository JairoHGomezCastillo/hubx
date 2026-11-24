package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	domain "hubx-api/internal/auth"
)

// signInRequest represents the POST /auth/sign-in request body.
type signInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// signInResponse represents the POST /auth/sign-in success response.
type signInResponse struct {
	Token string   `json:"token"`
	User  userInfo `json:"user"`
}

// userInfo is the user data returned to the client (no password hash).
type userInfo struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// errorResponse represents an error response.
type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Handler handles authentication HTTP requests.
type Handler struct {
	service domain.Service
}

// NewHandler creates a new authentication handler.
func NewHandler(service domain.Service) *Handler {
	return &Handler{service: service}
}

// SignIn handles POST /auth/sign-in requests.
func (h *Handler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req signInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "MISSING_REQUIRED_FIELDS", "Email and password are required")
		return
	}
	
	// Validate required fields (after trimming)
	email := strings.TrimSpace(req.Email)
	password := strings.TrimSpace(req.Password)
	
	if email == "" || password == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_REQUIRED_FIELDS", "Email and password are required")
		return
	}
	
	// Call service
	token, user, err := h.service.SignIn(r.Context(), email, password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			h.writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials")
			return
		}
		// Internal server error
		slog.Error("sign-in failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
		return
	}
	
	// Success
	resp := signInResponse{
		Token: token,
		User: userInfo{
			Email: user.Email,
			Role:  user.Role,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// writeError writes a JSON error response.
func (h *Handler) writeError(w http.ResponseWriter, status int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error:   errorCode,
		Message: message,
	})
}
