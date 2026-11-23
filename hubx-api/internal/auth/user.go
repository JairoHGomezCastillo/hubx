// Package auth implements authentication services for HUBX users.
package auth

import "time"

// User represents an authenticated HUBX user. Fields match the users table
// in migration 0001_baseline_schema.sql.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"-"` // Never serialize
	CreatedAt    time.Time `json:"created_at"`
}
