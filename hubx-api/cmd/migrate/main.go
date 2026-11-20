// migrate applies the pending SQL migrations of hubx-api and exits.
// It reads db.url like the API does (DB_URL in the environment) and is safe to run on every deploy.
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"hubx-api/internal/config"
	"hubx-api/internal/migrate"
	"hubx-api/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	if err := run(); err != nil {
		logger.Error("migration failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(".", os.Getenv)
	if err != nil {
		return err
	}
	url := cfg.Get("db.url")
	if url == "" {
		return errMissingDBURL
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	db, err := sql.Open("pgx", url)
	if err != nil {
		return err
	}
	defer db.Close()

	migs, err := migrate.Load(migrations.FS)
	if err != nil {
		return err
	}
	applied, err := migrate.Apply(ctx, db, migs)
	if err != nil {
		return err
	}
	slog.Info("migrations up to date", "applied", applied, "total", len(migs))
	return nil
}

type configError string

func (e configError) Error() string { return string(e) }

const errMissingDBURL = configError("db.url is empty: set DB_URL in the environment")
