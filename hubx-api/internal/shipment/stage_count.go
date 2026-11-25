package shipment

import (
	"context"
)

// StageCount represents the count of shipments in a single stage.
type StageCount struct {
	Stage Stage `json:"stage"`
	Count int   `json:"count"`
}

// StageCountReport is the aggregate of all six stages in journey order.
type StageCountReport struct {
	Stages []StageCount `json:"stages"`
	Total  int          `json:"total"`
}

// StageCounter counts shipments by stage and returns a fixed six-entry report
// in journey order, with absent stages zero-filled and the total computed.
type StageCounter interface {
	Count(ctx context.Context) (*StageCountReport, error)
}

type stageCounter struct {
	repo StageCountRepository
}

// NewStageCounter creates a new stage counter service.
func NewStageCounter(repo StageCountRepository) StageCounter {
	return &stageCounter{repo: repo}
}

// Count returns exactly six stage counts in journey order: Created, In flight,
// At destination airport, Assigned to partner, Out for delivery, Delivered.
// Stages absent from the repository map are zero-filled. Total is the sum.
func (s *stageCounter) Count(ctx context.Context) (*StageCountReport, error) {
	counts, err := s.repo.CountByStage(ctx)
	if err != nil {
		return nil, err
	}

	// Journey order from stage.go
	journeyOrder := []Stage{
		StageCreated,
		StageInFlight,
		StageAtDestinationAirport,
		StageAssignedToPartner,
		StageOutForDelivery,
		StageDelivered,
	}

	stages := make([]StageCount, 0, 6)
	total := 0

	for _, stage := range journeyOrder {
		count := counts[stage] // zero if absent (map default)
		stages = append(stages, StageCount{
			Stage: stage,
			Count: count,
		})
		total += count
	}

	return &StageCountReport{
		Stages: stages,
		Total:  total,
	}, nil
}
