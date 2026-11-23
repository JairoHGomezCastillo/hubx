package auth

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidCredentials is returned for both wrong password and unknown email
	// to prevent user enumeration (Constitution VI).
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// Service defines authentication business logic.
type Service interface {
	SignIn(ctx context.Context, email, password string) (string, *User, error)
}

// AuthService implements Service.
type AuthService struct {
	repo             Repository
	jwtSecret        string
	expirationHours  int
}

// NewAuthService creates a new authentication service.
func NewAuthService(repo Repository, jwtSecret string, expirationHours int) *AuthService {
	return &AuthService{
		repo:            repo,
		jwtSecret:       jwtSecret,
		expirationHours: expirationHours,
	}
}

// SignIn authenticates a user by email and password, returning a JWT token and user info.
// Returns ErrInvalidCredentials for both unknown email and wrong password (generic error
// to prevent user enumeration per spec requirement).
func (s *AuthService) SignIn(ctx context.Context, email, password string) (string, *User, error) {
	// Normalize email before lookup
	normalized := NormalizeEmail(email)
	
	user, err := s.repo.FindUserByEmail(ctx, normalized)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Compare against a dummy hash to prevent timing attacks that could
			// reveal whether the email exists (Constitution VI: prevent user enumeration)
			_ = bcrypt.CompareHashAndPassword(
				[]byte("$2a$12$D.Y8Bwuo1bnt3fmAWPujjuJu00X7sGR6bT5FDkfaXhQLjdvNf/PnK"),
				[]byte(password),
			)
			return "", nil, ErrInvalidCredentials
		}
		// Database error
		return "", nil, err
	}
	
	// Compare password with stored hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		// Wrong password - return same error as unknown email
		return "", nil, ErrInvalidCredentials
	}
	
	// Generate JWT token
	token, err := GenerateToken(user, s.jwtSecret, s.expirationHours)
	if err != nil {
		return "", nil, err
	}
	
	return token, user, nil
}
