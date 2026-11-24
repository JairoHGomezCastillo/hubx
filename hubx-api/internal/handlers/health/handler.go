// Package health expone el endpoint de salud del servicio hubx-api.
// Cada módulo del dominio se registra a sí mismo en el mux de la
// aplicación, manteniendo el acoplamiento del main al mínimo.
package health

import (
	"encoding/json"
	"net/http"
)

// healthResponse es el contrato del cuerpo de respuesta del endpoint.
type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// NewHandler devuelve el handler HTTP del módulo. Acepta únicamente GET;
// cualquier otro método se rechaza con 405 Method Not Allowed.
func NewHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, err := json.Marshal(healthResponse{Status: "ok", Service: "hubx-api"})
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(body)
	})
}

// Register monta las rutas del módulo en el mux usando el enrutamiento
// nativo por método de Go 1.22+ ("GET /health").
func Register(mux *http.ServeMux) {
	mux.Handle("GET /health", NewHandler())
}
