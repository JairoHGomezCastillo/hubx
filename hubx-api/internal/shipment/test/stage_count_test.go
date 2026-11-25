package test

import (
	"context"
	"errors"
	"testing"

	"hubx-api/internal/shipment"
	"hubx-api/internal/shipment/mocks"

	"go.uber.org/mock/gomock"
)

// HU09_Count_ReturnsAllSixStagesInJourneyOrder verifies that the counter
// returns exactly six stages in journey order: Created, In flight,
// At destination airport, Assigned to partner, Out for delivery, Delivered.
func TestHU09_Count_ReturnsAllSixStagesInJourneyOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockStageCountRepository(ctrl)
	counter := shipment.NewStageCounter(repo)

	// Repository returns a sparse map
	repo.EXPECT().
		CountByStage(gomock.Any()).
		Return(map[shipment.Stage]int{
			shipment.StageInFlight:       3,
			shipment.StageDelivered:      5,
			shipment.StageOutForDelivery: 2,
		}, nil)

	report, err := counter.Count(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify exactly six entries in journey order
	expected := []shipment.Stage{
		shipment.StageCreated,
		shipment.StageInFlight,
		shipment.StageAtDestinationAirport,
		shipment.StageAssignedToPartner,
		shipment.StageOutForDelivery,
		shipment.StageDelivered,
	}

	if len(report.Stages) != 6 {
		t.Fatalf("expected 6 stages, got %d", len(report.Stages))
	}

	for i, stage := range expected {
		if report.Stages[i].Stage != stage {
			t.Errorf("stage[%d]: expected %q, got %q", i, stage, report.Stages[i].Stage)
		}
	}
}

// HU09_Count_ZeroFillsAbsentStages verifies that stages absent from the
// repository map are filled with count=0.
func TestHU09_Count_ZeroFillsAbsentStages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockStageCountRepository(ctrl)
	counter := shipment.NewStageCounter(repo)

	// Repository returns only two stages
	repo.EXPECT().
		CountByStage(gomock.Any()).
		Return(map[shipment.Stage]int{
			shipment.StageInFlight:  3,
			shipment.StageDelivered: 5,
		}, nil)

	report, err := counter.Count(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify absent stages have count=0
	if report.Stages[0].Count != 0 { // Created
		t.Errorf("Created: expected count=0, got %d", report.Stages[0].Count)
	}
	if report.Stages[1].Count != 3 { // In flight
		t.Errorf("In flight: expected count=3, got %d", report.Stages[1].Count)
	}
	if report.Stages[2].Count != 0 { // At destination airport
		t.Errorf("At destination airport: expected count=0, got %d", report.Stages[2].Count)
	}
	if report.Stages[3].Count != 0 { // Assigned to partner
		t.Errorf("Assigned to partner: expected count=0, got %d", report.Stages[3].Count)
	}
	if report.Stages[4].Count != 0 { // Out for delivery
		t.Errorf("Out for delivery: expected count=0, got %d", report.Stages[4].Count)
	}
	if report.Stages[5].Count != 5 { // Delivered
		t.Errorf("Delivered: expected count=5, got %d", report.Stages[5].Count)
	}
}

// HU09_Count_ComputesTotalAsSum verifies that Total equals the sum of all counts.
func TestHU09_Count_ComputesTotalAsSum(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockStageCountRepository(ctrl)
	counter := shipment.NewStageCounter(repo)

	repo.EXPECT().
		CountByStage(gomock.Any()).
		Return(map[shipment.Stage]int{
			shipment.StageInFlight:       3,
			shipment.StageOutForDelivery: 2,
			shipment.StageDelivered:      5,
		}, nil)

	report, err := counter.Count(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedTotal := 3 + 2 + 5 // = 10
	if report.Total != expectedTotal {
		t.Errorf("expected total=%d, got %d", expectedTotal, report.Total)
	}
}

// HU09_Count_EmptyMapReturnsAllZeros verifies that an empty repository map
// returns six stages all with count=0 and total=0.
func TestHU09_Count_EmptyMapReturnsAllZeros(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockStageCountRepository(ctrl)
	counter := shipment.NewStageCounter(repo)

	// Repository returns empty map (no shipments)
	repo.EXPECT().
		CountByStage(gomock.Any()).
		Return(map[shipment.Stage]int{}, nil)

	report, err := counter.Count(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all counts are zero
	for i, stageCount := range report.Stages {
		if stageCount.Count != 0 {
			t.Errorf("stage[%d] %q: expected count=0, got %d", i, stageCount.Stage, stageCount.Count)
		}
	}

	if report.Total != 0 {
		t.Errorf("expected total=0, got %d", report.Total)
	}
}

// HU09_Count_PropagatesRepositoryError verifies that a repository error
// is returned unchanged.
func TestHU09_Count_PropagatesRepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockStageCountRepository(ctrl)
	counter := shipment.NewStageCounter(repo)

	expectedErr := errors.New("database connection failed")
	repo.EXPECT().
		CountByStage(gomock.Any()).
		Return(nil, expectedErr)

	_, err := counter.Count(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}
