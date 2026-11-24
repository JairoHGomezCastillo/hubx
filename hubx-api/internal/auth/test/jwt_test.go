package test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"hubx-api/internal/auth"
)

func TestHU01_GenerateToken(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	user := &auth.User{
		ID:    "550e8400-e29b-41d4-a716-446655440000",
		Email: "operator@hubx.co",
		Role:  "operator",
	}

	tests := []struct {
		name             string
		user             *auth.User
		secret           string
		expirationHours  int
	}{
		{
			name:            "generate token for operator",
			user:            user,
			secret:          secret,
			expirationHours: 1,
		},
		{
			name: "generate token for ground partner",
			user: &auth.User{
				ID:    "00000000-0000-4000-8000-000000000002",
				Email: "rapido@hubx.co",
				Role:  "ground partner",
			},
			secret:          secret,
			expirationHours: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := auth.GenerateToken(tt.user, tt.secret, tt.expirationHours)
			if err != nil {
				t.Fatalf("GenerateToken() error = %v", err)
			}
			if token == "" {
				t.Fatal("GenerateToken() returned empty token")
			}

			// Verify the token can be parsed and contains correct claims
			parsed, err := jwt.ParseWithClaims(token, &auth.Claims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(tt.secret), nil
			})
			if err != nil {
				t.Fatalf("Failed to parse generated token: %v", err)
			}

			claims, ok := parsed.Claims.(*auth.Claims)
			if !ok {
				t.Fatal("Failed to cast claims")
			}

			if claims.Sub != tt.user.ID {
				t.Errorf("claims.Sub = %q; want %q", claims.Sub, tt.user.ID)
			}
			if claims.Email != tt.user.Email {
				t.Errorf("claims.Email = %q; want %q", claims.Email, tt.user.Email)
			}
			if claims.Role != tt.user.Role {
				t.Errorf("claims.Role = %q; want %q", claims.Role, tt.user.Role)
			}

			// Verify expiration is approximately correct (within 10 seconds)
			expectedExp := time.Now().Add(time.Duration(tt.expirationHours) * time.Hour)
			actualExp := claims.ExpiresAt.Time
			diff := actualExp.Sub(expectedExp).Abs()
			if diff > 10*time.Second {
				t.Errorf("expiration time diff = %v; want < 10s", diff)
			}
		})
	}
}

func TestHU01_ValidateToken_Success(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	user := &auth.User{
		ID:    "550e8400-e29b-41d4-a716-446655440000",
		Email: "operator@hubx.co",
		Role:  "operator",
	}

	token, err := auth.GenerateToken(user, secret, 1)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := auth.ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.Sub != user.ID {
		t.Errorf("claims.Sub = %q; want %q", claims.Sub, user.ID)
	}
	if claims.Email != user.Email {
		t.Errorf("claims.Email = %q; want %q", claims.Email, user.Email)
	}
	if claims.Role != user.Role {
		t.Errorf("claims.Role = %q; want %q", claims.Role, user.Role)
	}
}

func TestHU01_ValidateToken_ExpiredToken(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	
	// Create an expired token (exp in the past)
	now := time.Now()
	claims := auth.Claims{
		Sub:   "550e8400-e29b-41d4-a716-446655440000",
		Email: "operator@hubx.co",
		Role:  "operator",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)), // Expired 1 hour ago
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to create expired token: %v", err)
	}

	_, err = auth.ValidateToken(tokenString, secret)
	if err != auth.ErrExpiredToken {
		t.Errorf("ValidateToken() error = %v; want ErrExpiredToken", err)
	}
}

func TestHU01_ValidateToken_InvalidSignature(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"
	wrongSecret := "wrong-secret-key-32-bytes-long!!"
	
	user := &auth.User{
		ID:    "550e8400-e29b-41d4-a716-446655440000",
		Email: "operator@hubx.co",
		Role:  "operator",
	}

	// Generate token with one secret
	token, err := auth.GenerateToken(user, secret, 1)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Try to validate with wrong secret
	_, err = auth.ValidateToken(token, wrongSecret)
	if err != auth.ErrInvalidToken {
		t.Errorf("ValidateToken() error = %v; want ErrInvalidToken", err)
	}
}

func TestHU01_ValidateToken_MalformedToken(t *testing.T) {
	secret := "test-secret-key-32-bytes-long!!"

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "not a JWT",
			token: "not-a-jwt-token",
		},
		{
			name:  "malformed JWT",
			token: "header.payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.ValidateToken(tt.token, secret)
			if err != auth.ErrInvalidToken {
				t.Errorf("ValidateToken() error = %v; want ErrInvalidToken", err)
			}
		})
	}
}
