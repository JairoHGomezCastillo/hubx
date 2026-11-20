package migrate

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"testing/fstest"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestLoad_SortsByVersionAndIgnoresNonSQL(t *testing.T) {
	fsys := fstest.MapFS{
		"0002_add_incidents.sql":   {Data: []byte("SELECT 2;")},
		"0001_baseline_schema.sql": {Data: []byte("SELECT 1;")},
		"embed.go":                 {Data: []byte("package migrations")},
		"README.md":                {Data: []byte("docs")},
	}

	got, err := Load(fsys)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d migrations, want 2", len(got))
	}
	if got[0].Version != "0001" || got[0].Name != "baseline_schema" || got[1].Version != "0002" {
		t.Fatalf("unexpected order or names: %+v", got)
	}
}

func TestLoad_ChecksumIgnoresLineEndings(t *testing.T) {
	lf, err := Load(fstest.MapFS{"0001_a.sql": {Data: []byte("SELECT 1;\nSELECT 2;\n")}})
	if err != nil {
		t.Fatal(err)
	}
	crlf, err := Load(fstest.MapFS{"0001_a.sql": {Data: []byte("SELECT 1;\r\nSELECT 2;\r\n")}})
	if err != nil {
		t.Fatal(err)
	}
	if lf[0].Checksum != crlf[0].Checksum {
		t.Fatal("checksum must not depend on line endings")
	}
}

func TestLoad_RejectsInvalidFiles(t *testing.T) {
	cases := map[string]fstest.MapFS{
		"bad name":          {"1_baseline.sql": {Data: []byte("SELECT 1;")}},
		"upper case name":   {"0001_Baseline.sql": {Data: []byte("SELECT 1;")}},
		"duplicate version": {"0001_a.sql": {Data: []byte("SELECT 1;")}, "0001_b.sql": {Data: []byte("SELECT 1;")}},
		"empty file":        {"0001_a.sql": {Data: []byte("  \n")}},
	}
	for name, fsys := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(fsys); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

// The tests below need a disposable PostgreSQL database: HUBX_TEST_DB_URL=postgres://... go test ./...
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("HUBX_TEST_DB_URL")
	if url == "" {
		t.Skip("HUBX_TEST_DB_URL not set")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	reset := `DROP TABLE IF EXISTS schema_migrations, migrate_probe`
	if _, err := db.Exec(reset); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec(reset) })
	return db
}

func TestApply_IsIdempotent(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	migs, err := Load(fstest.MapFS{
		"0001_probe.sql": {Data: []byte("CREATE TABLE migrate_probe (id int);\nINSERT INTO migrate_probe VALUES (1);")},
	})
	if err != nil {
		t.Fatal(err)
	}

	first, err := Apply(ctx, db, migs)
	if err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	second, err := Apply(ctx, db, migs)
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	if len(first) != 1 || len(second) != 0 {
		t.Fatalf("applied first=%v second=%v, want 1 then 0", first, second)
	}
	var rows int
	if err := db.QueryRow(`SELECT count(*) FROM migrate_probe`).Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("probe rows=%d err=%v, want 1", rows, err)
	}
}

func TestApply_FailedMigrationLeavesNoTrace(t *testing.T) {
	db := testDB(t)
	migs, err := Load(fstest.MapFS{
		"0001_probe.sql": {Data: []byte("CREATE TABLE migrate_probe (id int);\nSELECT * FROM missing_table;")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(context.Background(), db, migs); err == nil {
		t.Fatal("expected an error")
	}
	var exists bool
	if err := db.QueryRow(`SELECT to_regclass('migrate_probe') IS NOT NULL`).Scan(&exists); err != nil || exists {
		t.Fatalf("migrate_probe exists=%v err=%v, want rolled back", exists, err)
	}
}

func TestApply_RejectsEditedMigration(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	original, _ := Load(fstest.MapFS{"0001_probe.sql": {Data: []byte("CREATE TABLE migrate_probe (id int);")}})
	if _, err := Apply(ctx, db, original); err != nil {
		t.Fatal(err)
	}
	edited, _ := Load(fstest.MapFS{"0001_probe.sql": {Data: []byte("CREATE TABLE migrate_probe (id bigint);")}})
	if _, err := Apply(ctx, db, edited); !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("got %v, want ErrChecksumMismatch", err)
	}
}
