package test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	domain "hubx-api/internal/auth"
	"hubx-api/internal/handlers/auth"
	"hubx-api/internal/handlers/shipment"
	"hubx-api/internal/handlers/shipment/mocks"
	domainshipment "hubx-api/internal/shipment"

	"go.uber.org/mock/gomock"
)

// HU09_GetStageCounts_Operator_Returns200WithSixStagesInOrder verifies that
// an operator receives 200 with six stages in journey order and the total.
func TestHU09_GetStageCounts_Operator_Returns200WithSixStagesInOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	counter := mocks.NewMockStageCounter(ctrl)

	// Counter returns the report
	counter.EXPECT().
		Count(gomock.Any()).
		Return(&domainshipment.StageCountReport{
			Stages: []domainshipment.StageCount{
				{Stage: domainshipment.StageCreated, Count: 1},
				{Stage: domainshipment.StageInFlight, Count: 3},
				{Stage: domainshipment.StageAtDestinationAirport, Count: 0},
				{Stage: domainshipment.StageAssignedToPartner, Count: 0},
				{Stage: domainshipment.StageOutForDelivery, Count: 2},
				{Stage: domainshipment.StageDelivered, Count: 5},
			},
			Total: 11,
		}, nil)

	handler := shipment.NewStageCountsHandler(counter)

	req := httptest.NewRequest(http.MethodGet, "/shipments/stage-counts", nil)
	// Inject operator claims into context
	claims := &domain.Claims{
		Sub:   "operator-id",
		Email: "operator@hubx.co",
		Role:  "operator",
	}
	ctx := context.WithValue(req.Context(), auth.ClaimsContextKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify stages array
	stages, ok := resp["stages"].([]interface{})
	if !ok {
		t.Fatalf("expected stages array, got %T", resp["stages"])
	}

	if len(stages) != 6 {
		t.Errorf("expected 6 stages, got %d", len(stages))
	}

	// Verify journey order
	expectedStages := []string{
		"Created",
		"In flight",
		"At destination airport",
		"Assigned to partner",
		"Out for delivery",
		"Delivered",
	}

	for i, expected := range expectedStages {
		stage := stages[i].(map[string]interface{})
		if stage["stage"] != expected {
			t.Errorf("stages[%d]: expected %q, got %q", i, expected, stage["stage"])
		}
	}

	// Verify total field
	total, ok := resp["total"].(float64)
	if !ok {
		t.Fatalf("expected total to be a number, got %T", resp["total"])
	}
	if int(total) != 11 {
		t.Errorf("expected total=11, got %d", int(total))
	}
}

// HU09_GetStageCounts_GroundPartner_Returns403WithoutCallingCounter verifies
// that a ground partner receives 403 and the counter is never invoked.
func TestHU09_GetStageCounts_GroundPartner_Returns403WithoutCallingCounter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	counter := mocks.NewMockStageCounter(ctrl)
	// No EXPECT call - the counter should NOT be invoked

	handler := shipment.NewStageCountsHandler(counter)

	req := httptest.NewRequest(http.MethodGet, "/shipments/stage-counts", nil)
	claims := &domain.Claims{
		Sub:   "partner-id",
		Email: "rapido@hubx.co",
		Role:  "ground partner",
	}
	ctx := context.WithValue(req.Context(), auth.ClaimsContextKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["error"] != "FORBIDDEN" {
		t.Errorf("expected error=FORBIDDEN, got %v", resp["error"])
	}

	expectedMsg := "You do not have permission to see that information"
	if resp["message"] != expectedMsg {
		t.Errorf("expected message=%q, got %v", expectedMsg, resp["message"])
	}

	// Verify no stages or total in body
	if _, exists := resp["stages"]; exists {
		t.Error("403 body should not contain stages")
	}
	if _, exists := resp["total"]; exists {
		t.Error("403 body should not contain total")
	}
}

// HU09_GetStageCounts_Customer_Returns403WithoutCallingCounter verifies
// that a customer receives 403 and the counter is never invoked.
func TestHU09_GetStageCounts_Customer_Returns403WithoutCallingCounter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	counter := mocks.NewMockStageCounter(ctrl)
	// No EXPECT call - the counter should NOT be invoked

	handler := shipment.NewStageCountsHandler(counter)

	req := httptest.NewRequest(http.MethodGet, "/shipments/stage-counts", nil)
	claims := &domain.Claims{
		Sub:   "customer-id",
		Email: "ana@hubx.co",
		Role:  "customer",
	}
	ctx := context.WithValue(req.Context(), auth.ClaimsContextKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["error"] != "FORBIDDEN" {
		t.Errorf("expected error=FORBIDDEN, got %v", resp["error"])
	}
}

// HU09_GetStageCounts_CounterError_Returns500 verifies that when the counter
// returns an error, the handler maps it to 500 INTERNAL_SERVER_ERROR.
func TestHU09_GetStageCounts_CounterError_Returns500(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	counter := mocks.NewMockStageCounter(ctrl)

	expectedErr := errors.New("database error")
	counter.EXPECT().
		Count(gomock.Any()).
		Return(nil, expectedErr)

	handler := shipment.NewStageCountsHandler(counter)

	req := httptest.NewRequest(http.MethodGet, "/shipments/stage-counts", nil)
	claims := &domain.Claims{
		Sub:   "operator-id",
		Email: "operator@hubx.co",
		Role:  "operator",
	}
	ctx := context.WithValue(req.Context(), auth.ClaimsContextKey, claims)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["error"] != "INTERNAL_SERVER_ERROR" {
		t.Errorf("expected error=INTERNAL_SERVER_ERROR, got %v", resp["error"])
	}

	if resp["message"] != "An unexpected error occurred" {
		t.Errorf("expected generic message, got %v", resp["message"])
	}
}

// HU09_GetStageCounts_MissingClaims_Returns401 verifies that when no claims
// are in the context (middleware not applied), the handler returns 401.
func TestHU09_GetStageCounts_MissingClaims_Returns401(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	counter := mocks.NewMockStageCounter(ctrl)
	// No EXPECT call - should fail before calling counter

	handler := shipment.NewStageCountsHandler(counter)

	req := httptest.NewRequest(http.MethodGet, "/shipments/stage-counts", nil)
	// No claims in context

	w := httptest.NewRecorder()
	handler.Handle(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["error"] != "UNAUTHORIZED" {
		t.Errorf("expected error=UNAUTHORIZED, got %v", resp["error"])
	}
}
