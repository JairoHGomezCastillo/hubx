package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

var (
	// ErrUserNotFound is returned when the user does not exist.
	ErrUserNotFound = errors.New("user not found")
)

//go:generate go tool mockgen -source=repository.go -destination=mocks/repository_mock.go -package=mocks

// Repository defines persistence operations for users.
type Repository interface {
	FindUserByEmail(ctx context.Context, email string) (*User, error)
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// FindUserByEmail retrieves a user by email. The email is normalized
// (lowercase, trimmed) before lookup to match the database constraint.
// Returns ErrUserNotFound if the user does not exist.
func (r *PostgresRepository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	normalized := NormalizeEmail(email)
	
	query := `SELECT id, email, name, role, password_hash, created_at 
	          FROM users WHERE email = $1`
	
	var user User
	err := r.db.QueryRowContext(ctx, query, normalized).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.Role,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}

// NormalizeEmail returns the email in lowercase and trimmed of whitespace,
// matching the database CHECK constraint on the users.email column.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
