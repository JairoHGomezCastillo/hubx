package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when the JWT is malformed or signature is invalid.
	ErrInvalidToken = errors.New("invalid token")
	
	// ErrExpiredToken is returned when the JWT exp claim is in the past.
	ErrExpiredToken = errors.New("expired token")
)

// Claims represents the JWT payload for HUBX sessions.
type Claims struct {
	Sub   string `json:"sub"`   // User ID
	Email string `json:"email"` // User email
	Role  string `json:"role"`  // User role (operator, ground partner, customer)
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT for the given user, signed with HS256.
// The token expires after expirationHours.
func GenerateToken(user *User, secret string, expirationHours int) (string, error) {
	now := time.Now()
	claims := Claims{
		Sub:   user.ID,
		Email: user.Email,
		Role:  user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expirationHours) * time.Hour)),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken verifies the JWT signature and checks expiration.
// Returns the claims if valid, or an error (ErrInvalidToken, ErrExpiredToken).
func ValidateToken(tokenString, secret string) (*Claims, error) {
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
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	
	return claims, nil
}
