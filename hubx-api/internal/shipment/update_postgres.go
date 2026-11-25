package shipment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

//go:generate go tool mockgen -source=update_postgres.go -destination=mocks/postgres_mock.go -package=mocks

const (
	findShipmentByCodeSQL = `SELECT id, code, stage, customer_id, assigned_partner_id, created_by
		FROM shipments WHERE code = $1`

	findShipmentByIDSQL = `SELECT id, code, stage, customer_id, assigned_partner_id, created_by
		FROM shipments WHERE id = $1`

	findShipmentsByIDsSQL = `SELECT id, code, origin, destination, flight, weight_kg, stage, assigned_partner_id
		FROM shipments WHERE id = ANY($1)`

	isRegisteredPartnerSQL = `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND role = $2)`

	listGroundPartnersSQL = `SELECT id, name FROM users WHERE role = $1 ORDER BY name`

	partiesSQL = `SELECT id, name, role FROM users WHERE id = ANY($1)`

	userNameSQL = `SELECT name FROM users WHERE id = $1`
)

// DB is the injectable database handle used by the PostgreSQL repository.
type DB interface {
	QueryRowContext(ctx context.Context, query string, args ...any) Row
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
}

// Row is the single-row result of QueryRowContext.
type Row interface {
	Scan(dest ...any) error
}

// Rows is the multi-row result of QueryContext.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close() error
}

type sqlDB struct {
	db *sql.DB
}

// NewDB adapts a *sql.DB to the injectable DB interface.
func NewDB(db *sql.DB) DB {
	return sqlDB{db: db}
}

func (d sqlDB) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return d.db.QueryRowContext(ctx, query, args...)
}

func (d sqlDB) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return d.db.QueryContext(ctx, query, args...)
}

// PostgresRepository reads the shipment projections and the user directory
// from PostgreSQL. Stage and assigned partner are projections of the ledger;
// this repository never writes either of them.
type PostgresRepository struct {
	db DB
}

var (
	_ ShipmentRepository      = (*PostgresRepository)(nil)
	_ GroundPartnerRepository = (*PostgresRepository)(nil)
	_ PartyRepository         = (*PostgresRepository)(nil)
)

// Name(ctx context.Context, userID string) (string, error) on this type also
// satisfies the transport's PartyDirectory interface structurally, without
// the domain importing the handler package.

// NewPostgresRepository returns the PostgreSQL-backed repositories.
func NewPostgresRepository(db DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// FindByCode returns (nil, nil) when no shipment has this public code, so the
// caller can answer like it does for a shipment it cannot see.
func (r *PostgresRepository) FindByCode(ctx context.Context, code string) (*Shipment, error) {
	return scanShipment(r.db.QueryRowContext(ctx, findShipmentByCodeSQL, code))
}

// FindByID returns (nil, nil) when no shipment has this identifier.
func (r *PostgresRepository) FindByID(ctx context.Context, id string) (*Shipment, error) {
	return scanShipment(r.db.QueryRowContext(ctx, findShipmentByIDSQL, id))
}

// FindByIDs returns shipments for the given ids with their display fields
// (code, origin, destination, flight, weight, stage). Returns an empty non-nil
// slice when ids is empty or no shipments match. Missing ids are silently omitted.
func (r *PostgresRepository) FindByIDs(ctx context.Context, ids []string) ([]Shipment, error) {
	if len(ids) == 0 {
		return []Shipment{}, nil
	}

	rows, err := r.db.QueryContext(ctx, findShipmentsByIDsSQL, ids)
	if err != nil {
		return nil, fmt.Errorf("query shipments: %w", err)
	}
	defer rows.Close()

	shipments := make([]Shipment, 0)
	for rows.Next() {
		var id, code, origin, destination, flight, weightKg, stage string
		var assignedPartnerID sql.NullString

		err := rows.Scan(&id, &code, &origin, &destination, &flight, &weightKg, &stage, &assignedPartnerID)
		if err != nil {
			return nil, fmt.Errorf("scan shipment: %w", err)
		}

		shipment := Shipment{
			ID:          id,
			Code:        code,
			Origin:      origin,
			Destination: destination,
			Flight:      flight,
			WeightKg:    weightKg,
			Stage:       Stage(stage),
		}
		if assignedPartnerID.Valid {
			value := assignedPartnerID.String
			shipment.AssignedPartnerID = &value
		}
		shipments = append(shipments, shipment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate shipments: %w", err)
	}
	return shipments, nil
}

func scanShipment(row Row) (*Shipment, error) {
	var id, code, stage, customerID, createdBy string
	var assignedPartnerID sql.NullString

	err := row.Scan(&id, &code, &stage, &customerID, &assignedPartnerID, &createdBy)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read shipment: %w", err)
	}

	result := &Shipment{
		ID:         id,
		Code:       code,
		Stage:      Stage(stage),
		CustomerID: customerID,
		CreatedBy:  createdBy,
	}
	if assignedPartnerID.Valid {
		value := assignedPartnerID.String
		result.AssignedPartnerID = &value
	}
	return result, nil
}

// IsRegisteredPartner reports whether userID is a registered ground partner.
func (r *PostgresRepository) IsRegisteredPartner(ctx context.Context, userID string) (bool, error) {
	var registered bool
	if err := r.db.QueryRowContext(ctx, isRegisteredPartnerSQL, userID, string(RoleGroundPartner)).Scan(&registered); err != nil {
		return false, fmt.Errorf("check registered ground partner: %w", err)
	}
	return registered, nil
}

// List returns every registered ground partner in the repository's name order.
func (r *PostgresRepository) List(ctx context.Context) ([]GroundPartner, error) {
	rows, err := r.db.QueryContext(ctx, listGroundPartnersSQL, string(RoleGroundPartner))
	if err != nil {
		return nil, fmt.Errorf("list ground partners: %w", err)
	}
	defer rows.Close()

	partners := make([]GroundPartner, 0)
	for rows.Next() {
		var partner GroundPartner
		if err := rows.Scan(&partner.ID, &partner.Name); err != nil {
			return nil, fmt.Errorf("scan ground partner: %w", err)
		}
		partners = append(partners, partner)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ground partners: %w", err)
	}
	return partners, nil
}

// Name resolves the display name of a user. Errors propagate to the caller so
// the transport can decide how to render a party whose name is unavailable.
func (r *PostgresRepository) Name(ctx context.Context, userID string) (string, error) {
	var name string
	if err := r.db.QueryRowContext(ctx, userNameSQL, userID).Scan(&name); err != nil {
		return "", fmt.Errorf("read user name: %w", err)
	}
	return name, nil
}

// Parties resolves the name and role of multiple users in a single parameterized query.
// It returns a map keyed by ID; missing ids are silently omitted from the result.
func (r *PostgresRepository) Parties(ctx context.Context, ids []string) (map[string]Party, error) {
	if len(ids) == 0 {
		return make(map[string]Party), nil
	}

	rows, err := r.db.QueryContext(ctx, partiesSQL, ids)
	if err != nil {
		return nil, fmt.Errorf("query parties: %w", err)
	}
	defer rows.Close()

	parties := make(map[string]Party, len(ids))
	for rows.Next() {
		var party Party
		if err := rows.Scan(&party.ID, &party.Name, &party.Role); err != nil {
			return nil, fmt.Errorf("scan party: %w", err)
		}
		parties[party.ID] = party
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate parties: %w", err)
	}
	return parties, nil
}
