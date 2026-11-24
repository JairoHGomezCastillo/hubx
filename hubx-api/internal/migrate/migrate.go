// Package migrate applies plain-SQL migrations (NNNN_name.sql) to PostgreSQL in version order.
// It uses only database/sql: each pending migration runs in its own transaction together with its
// row in schema_migrations, so a failure leaves no trace. A session advisory lock serializes
// concurrent runners, and a checksum rejects migrations edited after being applied.
package migrate

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

// ErrChecksumMismatch means an already applied migration was edited; add a new migration instead.
var ErrChecksumMismatch = errors.New("applied migration was modified")

// lockKey identifies the HUBX migration advisory lock ("hubx" in ASCII).
const lockKey = 0x68756278

var fileName = regexp.MustCompile(`^(\d{4})_([a-z0-9_]+)\.sql$`)

// Migration is one SQL file ready to apply.
type Migration struct {
	Version  string
	Name     string
	SQL      string
	Checksum string
}

// Load reads every *.sql file at the root of fsys, validates its name and returns them sorted by version.
func Load(fsys fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	seen := map[string]string{}
	var migs []Migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		m := fileName.FindStringSubmatch(e.Name())
		if m == nil {
			return nil, fmt.Errorf("migration %q: name must match NNNN_lower_snake.sql", e.Name())
		}
		if prev, dup := seen[m[1]]; dup {
			return nil, fmt.Errorf("migrations %q and %q share version %s", prev, e.Name(), m[1])
		}
		seen[m[1]] = e.Name()

		raw, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		body := strings.ReplaceAll(string(raw), "\r\n", "\n")
		if strings.TrimSpace(body) == "" {
			return nil, fmt.Errorf("migration %q is empty", e.Name())
		}
		sum := sha256.Sum256([]byte(body))
		migs = append(migs, Migration{Version: m[1], Name: m[2], SQL: body, Checksum: hex.EncodeToString(sum[:])})
	}
	sort.Slice(migs, func(i, j int) bool { return migs[i].Version < migs[j].Version })
	return migs, nil
}

// Apply runs the migrations not yet recorded in schema_migrations and returns the versions it applied.
func Apply(ctx context.Context, db *sql.DB, migs []Migration) (applied []string, err error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, lockKey); err != nil {
		return nil, fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		if _, uerr := conn.ExecContext(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, lockKey); uerr != nil && err == nil {
			err = fmt.Errorf("release migration lock: %w", uerr)
		}
	}()

	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    text        PRIMARY KEY,
		name       text        NOT NULL,
		checksum   text        NOT NULL,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return nil, fmt.Errorf("create schema_migrations: %w", err)
	}

	done, err := appliedChecksums(ctx, conn)
	if err != nil {
		return nil, err
	}

	for _, m := range migs {
		if sum, ok := done[m.Version]; ok {
			if sum != m.Checksum {
				return applied, fmt.Errorf("%s_%s: %w", m.Version, m.Name, ErrChecksumMismatch)
			}
			continue
		}
		if err := applyOne(ctx, conn, m); err != nil {
			return applied, err
		}
		applied = append(applied, m.Version)
	}
	return applied, nil
}

func appliedChecksums(ctx context.Context, conn *sql.Conn) (map[string]string, error) {
	rows, err := conn.QueryContext(ctx, `SELECT version, checksum FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()
	done := map[string]string{}
	for rows.Next() {
		var v, sum string
		if err := rows.Scan(&v, &sum); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		done[v] = sum
	}
	return done, rows.Err()
}

func applyOne(ctx context.Context, conn *sql.Conn, m Migration) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s_%s: begin: %w", m.Version, m.Name, err)
	}
	defer func() { _ = tx.Rollback() }()

	// No arguments: the driver sends the file with the simple protocol, so it may hold several statements.
	if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
		return fmt.Errorf("%s_%s: %w", m.Version, m.Name, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, name, checksum) VALUES ($1, $2, $3)`,
		m.Version, m.Name, m.Checksum); err != nil {
		return fmt.Errorf("%s_%s: record: %w", m.Version, m.Name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s_%s: commit: %w", m.Version, m.Name, err)
	}
	return nil
}
