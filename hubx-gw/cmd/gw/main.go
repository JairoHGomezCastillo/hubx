// hubx-gw es el gateway de HUBX.
// Único punto de entrada del MFE: no contiene lógica de dominio,
// solo reenvía a la API y expone su propia salud.
package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"hubx-gw/internal/middleware"
	"hubx-gw/internal/proxy"
)

// env lee una variable de entorno con un valor por defecto seguro.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Verify JWT secret is set (required for JWT validation)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Error("JWT_SECRET environment variable is required")
		os.Exit(1)
	}

	// Verify the gateway shared secret is set (injected toward the API so the
	// API can reject any request that did not come through the gateway).
	gatewaySecret := os.Getenv("GATEWAY_SECRET")
	if gatewaySecret == "" {
		logger.Error("GATEWAY_SECRET environment variable is required")
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "hubx-gw",
		})
	})

	apiURL := env("API_URL", "http://localhost:8081")
	proxy.Register(mux, apiURL, gatewaySecret)

	// Apply JWT validation middleware to all routes except excluded paths
	excludedPaths := map[string]bool{
		"/health":           true, // Gateway health check
		"/api/health":       true, // API health check (proxied)
		"/api/auth/sign-in": true, // Sign-in endpoint
	}

	jwtMiddleware := middleware.JWTValidation(jwtSecret, excludedPaths)

	addr := ":" + env("PORT", "8080")
	srv := &http.Server{
		Addr:    addr,
		Handler: jwtMiddleware(mux),
		// Mitigación Slowloris; requisito de gosec.
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("hubx-gw escuchando", "addr", addr, "api_url", apiURL)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("servidor finalizado con error", "err", err)
		os.Exit(1)
	}
}
