package shipment

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

//go:generate go tool mockgen -source=repository.go -destination=mocks/repository_mock.go -package=mocks

// Customer represents a user looked up for shipment creation.
type Customer struct {
	ID    string
	Email string
	Name  string
	Role  string
}

// Repository defines data access for shipments.
type Repository interface {
	FindUserByEmail(ctx context.Context, email string) (*Customer, error)
	Insert(ctx context.Context, shipment *Shipment, createdBy, customerID string) error
}

// PgRepository implements Repository using PostgreSQL.
type PgRepository struct {
	db *sql.DB
}

// NewPgRepository creates a new PostgreSQL repository.
func NewPgRepository(db *sql.DB) *PgRepository {
	return &PgRepository{db: db}
}

// FindUserByEmail looks up a user by their normalized email.
func (r *PgRepository) FindUserByEmail(ctx context.Context, email string) (*Customer, error) {
	query := `SELECT id, email, name, role FROM users WHERE email = $1`

	var c Customer
	err := r.db.QueryRowContext(ctx, query, email).Scan(&c.ID, &c.Email, &c.Name, &c.Role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}

	return &c, nil
}

// Insert inserts a new shipment into the database. Returns ErrDuplicateCode on
// unique constraint violation for the shipments_code_key constraint, or other database errors.
// Updates shipment.CreatedAt on success.
func (r *PgRepository) Insert(ctx context.Context, shipment *Shipment, createdBy, customerID string) error {
	query := `
		INSERT INTO shipments (code, origin, destination, flight, weight_kg, customer_id, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		shipment.Code,
		shipment.Origin,
		shipment.Destination,
		shipment.Flight,
		shipment.WeightKg,
		customerID,
		createdBy,
	).Scan(&shipment.CreatedAt)

	if err != nil {
		// Check for unique constraint violation on the shipments_code_key constraint
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "shipments_code_key" {
			return ErrDuplicateCode
		}
		return err
	}

	return nil
}
