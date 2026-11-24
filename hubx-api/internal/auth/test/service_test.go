package auth_test

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"go.uber.org/mock/gomock"

	"hubx-api/internal/auth"
	"hubx-api/internal/auth/mocks"
)

func TestHU01_SignIn_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a user with bcrypt hash of "test-password"
	hash, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}

	user := &auth.User{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		Email:        "operator@hubx.co",
		Name:         "Laura Operator",
		Role:         "operator",
		PasswordHash: string(hash),
	}

	repo := mocks.NewMockRepository(ctrl)
	repo.EXPECT().
		FindUserByEmail(gomock.Any(), "operator@hubx.co").
		Return(user, nil)

	service := auth.NewAuthService(repo, "test-secret", 1)

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

func TestHU01_SignIn_EmailNormalization(t *testing.T) {
	// Create a user with lowercase email
	hash, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}

	user := &auth.User{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		Email:        "ana@hubx.co",
		Name:         "Ana Customer",
		Role:         "customer",
		PasswordHash: string(hash),
	}

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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockRepository(ctrl)
			repo.EXPECT().
				FindUserByEmail(gomock.Any(), "ana@hubx.co").
				Return(user, nil)

			service := auth.NewAuthService(repo, "test-secret", 1)

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

func TestHU01_SignIn_WrongPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}

	user := &auth.User{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		Email:        "operator@hubx.co",
		Name:         "Laura Operator",
		Role:         "operator",
		PasswordHash: string(hash),
	}

	repo := mocks.NewMockRepository(ctrl)
	repo.EXPECT().
		FindUserByEmail(gomock.Any(), "operator@hubx.co").
		Return(user, nil)

	service := auth.NewAuthService(repo, "test-secret", 1)

	token, returnedUser, err := service.SignIn(context.Background(), "operator@hubx.co", "wrong-password")
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("SignIn() error = %v; want ErrInvalidCredentials", err)
	}
	if token != "" {
		t.Errorf("SignIn() token = %q; want empty", token)
	}
	if returnedUser != nil {
		t.Errorf("SignIn() user = %v; want nil", returnedUser)
	}
}

func TestHU01_SignIn_UnknownEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockRepository(ctrl)
	repo.EXPECT().
		FindUserByEmail(gomock.Any(), "nobody@hubx.co").
		Return(nil, auth.ErrUserNotFound)

	service := auth.NewAuthService(repo, "test-secret", 1)

	token, returnedUser, err := service.SignIn(context.Background(), "nobody@hubx.co", "any-password")
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("SignIn() error = %v; want ErrInvalidCredentials", err)
	}
	if token != "" {
		t.Errorf("SignIn() token = %q; want empty", token)
	}
	if returnedUser != nil {
		t.Errorf("SignIn() user = %v; want nil", returnedUser)
	}
}

func TestHU01_SignIn_GenericErrorMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Verify that wrong password and unknown email return the same error
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}

	user := &auth.User{
		ID:           "550e8400-e29b-41d4-a716-446655440000",
		Email:        "operator@hubx.co",
		Name:         "Laura Operator",
		Role:         "operator",
		PasswordHash: string(hash),
	}

	// Test wrong password
	repo1 := mocks.NewMockRepository(ctrl)
	repo1.EXPECT().
		FindUserByEmail(gomock.Any(), "operator@hubx.co").
		Return(user, nil)
	service1 := auth.NewAuthService(repo1, "test-secret", 1)
	_, _, err1 := service1.SignIn(context.Background(), "operator@hubx.co", "wrong-password")
	
	// Test unknown email
	repo2 := mocks.NewMockRepository(ctrl)
	repo2.EXPECT().
		FindUserByEmail(gomock.Any(), "nobody@hubx.co").
		Return(nil, auth.ErrUserNotFound)
	service2 := auth.NewAuthService(repo2, "test-secret", 1)
	_, _, err2 := service2.SignIn(context.Background(), "nobody@hubx.co", "any-password")

	// Both must return ErrInvalidCredentials
	if !errors.Is(err1, auth.ErrInvalidCredentials) {
		t.Errorf("Wrong password error = %v; want ErrInvalidCredentials", err1)
	}
	if !errors.Is(err2, auth.ErrInvalidCredentials) {
		t.Errorf("Unknown email error = %v; want ErrInvalidCredentials", err2)
	}
	if err1.Error() != err2.Error() {
		t.Errorf("Error messages differ: %q vs %q; want identical", err1, err2)
	}
}

func TestHU01_SignIn_GroundPartnerRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test with "ground partner" role (with space, per implement clarification)
	hash, err := bcrypt.GenerateFromPassword([]byte("test-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}

	user := &auth.User{
		ID:           "00000000-0000-4000-8000-000000000002",
		Email:        "rapido@hubx.co",
		Name:         "Transportes Rápido",
		Role:         "ground partner", // With space
		PasswordHash: string(hash),
	}

	repo := mocks.NewMockRepository(ctrl)
	repo.EXPECT().
		FindUserByEmail(gomock.Any(), "rapido@hubx.co").
		Return(user, nil)

	service := auth.NewAuthService(repo, "test-secret", 1)

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
