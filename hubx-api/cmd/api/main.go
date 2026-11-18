// hubx-api es el servicio de dominio de HUBX.
// Expone la salud del servicio y, en fases posteriores, los módulos del negocio.
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

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

	mux := http.NewServeMux()
	health.Register(mux)

	addr := ":" + port()
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
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
