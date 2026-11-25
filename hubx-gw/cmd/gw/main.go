// Package main is the HUBX gateway.
// Single entry point of the MFE: it holds no domain logic, it only forwards to
// the API and exposes its own health.
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"hubx-gw/internal/config"
	"hubx-gw/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Invalid configuration (limits, origins, missing secrets) stops the start.
	// The error names keys only: it never prints values or secrets.
	cfg, err := config.Load(".", os.Getenv)
	if err != nil {
		logger.Error("invalid configuration", "error", err.Error())
		os.Exit(1)
	}

	addr := ":" + cfg.Port
	srv := &http.Server{
		Addr:    addr,
		Handler: server.New(cfg, time.Now),
		// Slowloris mitigation; gosec requirement.
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("hubx-gw listening", "addr", addr, "api_url", cfg.APIURL,
		"requests_per_minute", cfg.RequestsPerMinute,
		"sign_in_requests_per_minute", cfg.SignInRequestsPerMinute,
		"allowed_origins", len(cfg.AllowedOrigins))
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server stopped with error", "error", err.Error())
		os.Exit(1)
	}
}
