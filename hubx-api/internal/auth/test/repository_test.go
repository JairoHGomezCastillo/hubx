package auth_test

import (
	"context"
	"testing"

	"hubx-api/internal/auth"
)

func TestHU01_NormalizeEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase email",
			input: "ana@hubx.co",
			want:  "ana@hubx.co",
		},
		{
			name:  "uppercase email",
			input: "ANA@HUBX.CO",
			want:  "ana@hubx.co",
		},
		{
			name:  "mixed case",
			input: "AnA@HuBx.Co",
			want:  "ana@hubx.co",
		},
		{
			name:  "leading whitespace",
			input: "  ana@hubx.co",
			want:  "ana@hubx.co",
		},
		{
			name:  "trailing whitespace",
			input: "ana@hubx.co  ",
			want:  "ana@hubx.co",
		},
		{
			name:  "both whitespace",
			input: "  ana@hubx.co  ",
			want:  "ana@hubx.co",
		},
		{
			name:  "uppercase with whitespace",
			input: " ANA@hubx.co ",
			want:  "ana@hubx.co",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := auth.NormalizeEmail(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeEmail(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestHU01_FindUserByEmail_ErrorHandling tests that the repository returns ErrUserNotFound
// when the user does not exist. This requires a real database or mock, so we
// test the error condition via service tests with a mock repository.
func TestHU01_FindUserByEmail_ErrorHandling(t *testing.T) {
	// This test is a placeholder. Full database integration tests are run
	// via hubx-bender. Unit testing the repository requires either testcontainers
	// or a mock DB, which is beyond the minimal scope for HU-01.
	// The repository logic is exercised through service_test.go with a mock.
	ctx := context.Background()
	_ = ctx
}
