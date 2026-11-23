package auth

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// mockRepository implements Repository for testing.
type mockRepository struct {
	users map[string]*User
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		users: make(map[string]*User),
	}
}

func (m *mockRepository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	normalized := NormalizeEmail(email)
	user, ok := m.users[normalized]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (m *mockRepository) addUser(user *User) {
	m.users[NormalizeEmail(user.Email)] = user
}

func TestSignIn_Success(t *testing.T) {
	// Create a user with bcrypt hash of "test-password"
	hash, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}

	user := &User{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		Email:        "operator@hubx.co",
		Name:         "Laura Operator",
		Role:         "operator",
		PasswordHash: string(hash),
	}

	repo := newMockRepository()
	repo.addUser(user)

	service := NewAuthService(repo, "test-secret", 1)

	token, returnedUser, err := service.SignIn(context.Background(), "operator@hubx.co", "test-password")
	if err != nil {
		t.Fatalf("SignIn() error = %v; want nil", err)
	}
	if token == "" {
		t.Error("SignIn() returned empty token")
	}
	if returnedUser.Email != user.Email {
		t.Errorf("SignIn() user.Email = %q; want %q", returnedUser.Email, user.Email)
	}
	if returnedUser.Role != user.Role {
		t.Errorf("SignIn() user.Role = %q; want %q", returnedUser.Role, user.Role)
	}
}

func TestSignIn_EmailNormalization(t *testing.T) {
	// Create a user with lowercase email
	hash, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}

	user := &User{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		Email:        "ana@hubx.co",
		Name:         "Ana Customer",
		Role:         "customer",
		PasswordHash: string(hash),
	}

	repo := newMockRepository()
	repo.addUser(user)

	service := NewAuthService(repo, "test-secret", 1)

	tests := []struct {
		name  string
		email string
	}{
		{
			name:  "uppercase",
			email: "ANA@HUBX.CO",
		},
		{
			name:  "mixed case",
			email: "AnA@HuBx.Co",
		},
		{
			name:  "with leading space",
			email: " ana@hubx.co",
		},
		{
			name:  "with trailing space",
			email: "ana@hubx.co ",
		},
		{
			name:  "with both spaces",
			email: "  ana@hubx.co  ",
		},
		{
			name:  "uppercase with spaces",
			email: " ANA@hubx.co ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, returnedUser, err := service.SignIn(context.Background(), tt.email, "test-password")
			if err != nil {
				t.Fatalf("SignIn(%q) error = %v; want nil", tt.email, err)
			}
			if token == "" {
				t.Error("SignIn() returned empty token")
			}
			if returnedUser.Email != user.Email {
				t.Errorf("SignIn() user.Email = %q; want %q", returnedUser.Email, user.Email)
			}
		})
	}
}

func TestSignIn_WrongPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}

	user := &User{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		Email:        "operator@hubx.co",
		Name:         "Laura Operator",
		Role:         "operator",
		PasswordHash: string(hash),
	}

	repo := newMockRepository()
	repo.addUser(user)

	service := NewAuthService(repo, "test-secret", 1)

	token, returnedUser, err := service.SignIn(context.Background(), "operator@hubx.co", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("SignIn() error = %v; want ErrInvalidCredentials", err)
	}
	if token != "" {
		t.Errorf("SignIn() token = %q; want empty", token)
	}
	if returnedUser != nil {
		t.Errorf("SignIn() user = %v; want nil", returnedUser)
	}
}

func TestSignIn_UnknownEmail(t *testing.T) {
	repo := newMockRepository()
	service := NewAuthService(repo, "test-secret", 1)

	token, returnedUser, err := service.SignIn(context.Background(), "nobody@hubx.co", "any-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("SignIn() error = %v; want ErrInvalidCredentials", err)
	}
	if token != "" {
		t.Errorf("SignIn() token = %q; want empty", token)
	}
	if returnedUser != nil {
		t.Errorf("SignIn() user = %v; want nil", returnedUser)
	}
}

func TestSignIn_GenericErrorMessage(t *testing.T) {
	// Verify that wrong password and unknown email return the same error
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}

	user := &User{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		Email:        "operator@hubx.co",
		Name:         "Laura Operator",
		Role:         "operator",
		PasswordHash: string(hash),
	}

	repo := newMockRepository()
	repo.addUser(user)

	service := NewAuthService(repo, "test-secret", 1)

	// Wrong password error
	_, _, err1 := service.SignIn(context.Background(), "operator@hubx.co", "wrong-password")
	
	// Unknown email error
	_, _, err2 := service.SignIn(context.Background(), "nobody@hubx.co", "any-password")

	// Both must return ErrInvalidCredentials
	if !errors.Is(err1, ErrInvalidCredentials) {
		t.Errorf("Wrong password error = %v; want ErrInvalidCredentials", err1)
	}
	if !errors.Is(err2, ErrInvalidCredentials) {
		t.Errorf("Unknown email error = %v; want ErrInvalidCredentials", err2)
	}
	if err1.Error() != err2.Error() {
		t.Errorf("Error messages differ: %q vs %q; want identical", err1, err2)
	}
}

func TestSignIn_GroundPartnerRole(t *testing.T) {
	// Test with "ground partner" role (with space, per implement clarification)
	hash, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}

	user := &User{
		ID:           "00000000-0000-4000-8000-000000000002",
		Email:        "rapido@hubx.co",
		Name:         "Transportes Rápido",
		Role:         "ground partner", // With space
		PasswordHash: string(hash),
	}

	repo := newMockRepository()
	repo.addUser(user)

	service := NewAuthService(repo, "test-secret", 1)

	token, returnedUser, err := service.SignIn(context.Background(), "rapido@hubx.co", "test-password")
	if err != nil {
		t.Fatalf("SignIn() error = %v; want nil", err)
	}
	if token == "" {
		t.Error("SignIn() returned empty token")
	}
	if returnedUser.Role != "ground partner" {
		t.Errorf("SignIn() user.Role = %q; want %q", returnedUser.Role, "ground partner")
	}
}
