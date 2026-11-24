package shipment

import (
	"database/sql"
	"net/http"

	domain "hubx-api/internal/shipment"
)

// Register registers shipment routes on the given mux.
// Returns the handler for testing purposes.
func Register(mux *http.ServeMux, db *sql.DB, requireAuth func(http.Handler) http.Handler) *Handler {
	// Initialize repository and service
	repo := domain.NewPgRepository(db)
	codeGen := domain.NewCryptoCodeGenerator()
	service := domain.NewShipmentService(repo, codeGen)

	handler := NewHandler(service)

	// Register protected route
	mux.Handle("POST /shipments", requireAuth(http.HandlerFunc(handler.Create)))

	return handler
}
