package ledger

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

//go:generate go tool mockgen -source=postgres.go -destination=mocks/postgres_mock.go -package=mocks

const (
	lockShipmentSQL = `SELECT id FROM shipments WHERE id = $1 FOR UPDATE`

	lastUpdateSQL = `SELECT id, sequence, stage, responsible_party_id,
		assigned_partner_id, recorded_at, previous_fingerprint, fingerprint
		FROM updates WHERE shipment_id = $1 ORDER BY sequence DESC LIMIT 1`

	listUpdatesSQL = `SELECT id, shipment_id, sequence, stage, responsible_party_id,
		assigned_partner_id, recorded_at, previous_fingerprint, fingerprint
		FROM updates WHERE shipment_id = $1 ORDER BY sequence ASC`

	insertUpdateSQL = `INSERT INTO updates
		(shipment_id, sequence, stage, responsible_party_id, assigned_partner_id,
		 recorded_at, previous_fingerprint, fingerprint)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`

	// An assigned partner is never replaced: a later update passes a null partner
	// and COALESCE keeps the one stored by the assignment.
	updateProjectionSQL = `UPDATE shipments SET stage = $1,
		assigned_partner_id = COALESCE($2, assigned_partner_id) WHERE id = $3`
)

// DB is the injectable database handle used by the PostgreSQL repository.
type DB interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
	QueryRowContext(ctx context.Context, query string, args ...any) Row
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
}

// Tx is the transactional handle needed to append an update atomically.
type Tx interface {
	QueryRowContext(ctx context.Context, query string, args ...any) Row
	ExecContext(ctx context.Context, query string, args ...any) (Result, error)
	Commit() error
	Rollback() error
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

// Result is the outcome of the projection write.
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

type sqlDB struct {
	db *sql.DB
}

// NewDB adapts a *sql.DB to the injectable DB interface.
func NewDB(db *sql.DB) DB {
	return sqlDB{db: db}
}

func (d sqlDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	tx, err := d.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return sqlTx{tx: tx}, nil
}

func (d sqlDB) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return d.db.QueryRowContext(ctx, query, args...)
}

func (d sqlDB) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return d.db.QueryContext(ctx, query, args...)
}

type sqlTx struct {
	tx *sql.Tx
}

func (t sqlTx) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

func (t sqlTx) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t sqlTx) Commit() error {
	return t.tx.Commit()
}

func (t sqlTx) Rollback() error {
	return t.tx.Rollback()
}

type postgresRepository struct {
	db DB
}

var _ LedgerRepository = (*postgresRepository)(nil)

// NewPostgresRepository returns the insert-only LedgerRepository backed by db.
func NewPostgresRepository(db DB) LedgerRepository {
	return &postgresRepository{db: db}
}

// Append locks the shipment row, rechecks the expected chain position and inserts
// one update in a single transaction. Existing updates are never written again.
func (r *postgresRepository) Append(ctx context.Context, in AppendInput) (*Update, error) {
	if err := ValidateAppendInput(in); err != nil {
		return nil, err
	}
	// HU-05: Truncate to microseconds before computing the fingerprint, so the
	// stored instant and the recomputed fingerprint match (PostgreSQL timestamptz
	// keeps microseconds while Go clocks have nanoseconds).
	recordedAt := in.RecordedAt.UTC().Truncate(time.Microsecond)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin append transaction: %w", mapPgError(err))
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var lockedShipmentID string
	if err := tx.QueryRowContext(ctx, lockShipmentSQL, in.ShipmentID).Scan(&lockedShipmentID); err != nil {
		return nil, fmt.Errorf("lock shipment: %w", mapPgError(err))
	}

	last, err := scanLastUpdate(tx.QueryRowContext(ctx, lastUpdateSQL, in.ShipmentID))
	if err != nil {
		return nil, err
	}
	if err := CheckSequence(in.Sequence, last); err != nil {
		return nil, err
	}
	if err := CheckPredecessor(in, last); err != nil {
		return nil, err
	}

	input := in
	input.RecordedAt = recordedAt
	fingerprint := Fingerprint(input)

	var id int64
	err = tx.QueryRowContext(ctx, insertUpdateSQL,
		input.ShipmentID,
		input.Sequence,
		input.Stage,
		input.ResponsiblePartyID,
		nullable(input.AssignedPartnerID),
		recordedAt,
		nullable(input.PreviousFingerprint),
		fingerprint,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("insert update: %w", mapPgError(err))
	}

	if _, err := tx.ExecContext(ctx, updateProjectionSQL,
		input.Stage, nullable(input.AssignedPartnerID), input.ShipmentID); err != nil {
		return nil, fmt.Errorf("update shipment projection: %w", mapPgError(err))
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit append transaction: %w", mapPgError(err))
	}
	committed = true

	return &Update{
		ID:                  id,
		ShipmentID:          input.ShipmentID,
		Sequence:            input.Sequence,
		Stage:               input.Stage,
		ResponsiblePartyID:  input.ResponsiblePartyID,
		AssignedPartnerID:   input.AssignedPartnerID,
		RecordedAt:          recordedAt,
		PreviousFingerprint: input.PreviousFingerprint,
		Fingerprint:         fingerprint,
	}, nil
}

// Last returns the newest stored update, or nil when the chain is empty.
func (r *postgresRepository) Last(ctx context.Context, shipmentID string) (*Update, error) {
	return scanLastUpdate(r.db.QueryRowContext(ctx, lastUpdateSQL, shipmentID))
}

// List returns every update of the shipment ordered by sequence ascending (oldest first).
// It returns an empty non-nil slice when no updates exist.
func (r *postgresRepository) List(ctx context.Context, shipmentID string) ([]Update, error) {
	rows, err := r.db.QueryContext(ctx, listUpdatesSQL, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("query updates: %w", mapPgError(err))
	}
	defer rows.Close()

	updates := make([]Update, 0)
	for rows.Next() {
		var id int64
		var shipmentID string
		var sequence int
		var stage, responsiblePartyID, fingerprint string
		var assignedPartnerID, previousFingerprint sql.NullString
		var recordedAt time.Time

		err := rows.Scan(&id, &shipmentID, &sequence, &stage, &responsiblePartyID,
			&assignedPartnerID, &recordedAt, &previousFingerprint, &fingerprint)
		if err != nil {
			return nil, fmt.Errorf("scan update: %w", mapPgError(err))
		}

		update := Update{
			ID:                 id,
			ShipmentID:         shipmentID,
			Sequence:           sequence,
			Stage:              stage,
			ResponsiblePartyID: responsiblePartyID,
			RecordedAt:         recordedAt.UTC(),
			Fingerprint:        fingerprint,
		}
		if assignedPartnerID.Valid {
			value := assignedPartnerID.String
			update.AssignedPartnerID = &value
		}
		if previousFingerprint.Valid {
			value := previousFingerprint.String
			update.PreviousFingerprint = &value
		}
		updates = append(updates, update)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate updates: %w", mapPgError(err))
	}
	return updates, nil
}

func scanLastUpdate(row Row) (*Update, error) {
	var id int64
	var sequence int
	var stage, responsiblePartyID, fingerprint string
	var assignedPartnerID, previousFingerprint sql.NullString
	var recordedAt time.Time

	err := row.Scan(&id, &sequence, &stage, &responsiblePartyID,
		&assignedPartnerID, &recordedAt, &previousFingerprint, &fingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read last update: %w", mapPgError(err))
	}

	update := &Update{
		ID:                 id,
		Sequence:           sequence,
		Stage:              stage,
		ResponsiblePartyID: responsiblePartyID,
		RecordedAt:         recordedAt.UTC(),
		Fingerprint:        fingerprint,
	}
	if assignedPartnerID.Valid {
		value := assignedPartnerID.String
		update.AssignedPartnerID = &value
	}
	if previousFingerprint.Valid {
		value := previousFingerprint.String
		update.PreviousFingerprint = &value
	}
	return update, nil
}

func nullable(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func mapPgError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "40001", "40P01":
			return fmt.Errorf("%w: %s", ErrConcurrentUpdate, pgErr.Code)
		}
	}
	return err
}
