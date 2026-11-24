// hubx-api es el servicio de dominio de HUBX.
// Expone la salud del servicio y, en fases posteriores, los módulos del negocio.
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"hubx-api/internal/config"
	"hubx-api/internal/handlers/auth"
	"hubx-api/internal/health"
)

// port lee el puerto desde el entorno con un valor por defecto seguro.
func port() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8081"
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load(".", os.Getenv)
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Verify JWT secret is set (required for auth)
	jwtSecret := cfg.Get("jwt.secret")
	if jwtSecret == "" {
		logger.Error("JWT_SECRET environment variable is required")
		os.Exit(1)
	}

	// Verify the gateway shared secret is set (required: the API only accepts
	// requests that come through the gateway, which injects this secret).
	gatewaySecret := cfg.Get("gateway.secret")
	if gatewaySecret == "" {
		logger.Error("GATEWAY_SECRET environment variable is required")
		os.Exit(1)
	}

	// Connect to database
	dbURL := cfg.Get("db.url")
	if dbURL == "" {
		logger.Error("DB_URL environment variable is required")
		os.Exit(1)
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Verify database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		logger.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	health.Register(mux)

	// Register auth module and get middleware for protected routes
	authMiddleware := auth.Register(mux, db, auth.Config{
		JWTSecret:       jwtSecret,
		ExpirationHours: cfg.GetInt("jwt.expiration.hours", 1),
	})

	// Future modules will use authMiddleware to protect their routes
	_ = authMiddleware

	// The gateway shared secret guards every route: all legitimate traffic
	// enters through the gateway, which injects X-Gateway-Secret.
	handler := auth.RequireGatewaySecret(gatewaySecret)(mux)

	addr := ":" + port()
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
		// ReadHeaderTimeout mitiga ataques Slowloris: no se acepta una
		// conexión que nunca termina de enviar sus cabeceras. gosec lo exige.
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("hubx-api escuchando", "addr", addr)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("servidor finalizado con error", "err", err)
		os.Exit(1)
	}
}
