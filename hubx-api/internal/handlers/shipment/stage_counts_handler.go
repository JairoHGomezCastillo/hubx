package shipment

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"hubx-api/internal/handlers/auth"
	domain "hubx-api/internal/shipment"
)

//go:generate go tool mockgen -source=stage_counts_handler.go -destination=mocks/stage_counter_mock.go -package=mocks

// StageCounter is the domain service interface for counting shipments by stage.
type StageCounter interface {
	Count(ctx context.Context) (*domain.StageCountReport, error)
}

// stageCountsHandler handles GET /shipments/stage-counts requests.
type stageCountsHandler struct {
	counter StageCounter
}

// NewStageCountsHandler creates a new stage counts handler.
func NewStageCountsHandler(counter StageCounter) *stageCountsHandler {
	return &stageCountsHandler{counter: counter}
}

// stageCountsResponse is the JSON body for a successful 200 response.
type stageCountsResponse struct {
	Stages []stageCountItem `json:"stages"`
	Total  int              `json:"total"`
}

type stageCountItem struct {
	Stage string `json:"stage"`
	Count int    `json:"count"`
}

// Handle handles GET /shipments/stage-counts.
// Authorization: operator-only. Non-operators receive 403 with no totals.
func (h *stageCountsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Extract claims from context (injected by RequireAuth middleware)
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header required")
		return
	}

	// Check role BEFORE calling the counter (plan D4, contract validation order)
	if claims.Role != "operator" {
		writeJSONError(w, http.StatusForbidden, "FORBIDDEN", "You do not have permission to see that information")
		return
	}

	// Call the counter service
	report, err := h.counter.Count(r.Context())
	if err != nil {
		slog.Error("stage counts failed", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
		return
	}

	// Build response in the contract format
	stages := make([]stageCountItem, len(report.Stages))
	for i, sc := range report.Stages {
		stages[i] = stageCountItem{
			Stage: string(sc.Stage),
			Count: sc.Count,
		}
	}

	resp := stageCountsResponse{
		Stages: stages,
		Total:  report.Total,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, status int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error:   errorCode,
		Message: message,
	})
}
