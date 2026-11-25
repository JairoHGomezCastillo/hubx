package shipment

import "context"

//go:generate go tool mockgen -source=stage_count_repository.go -destination=mocks/stage_count_repository_mock.go -package=mocks

// StageCountRepository retrieves shipment counts grouped by stage.
type StageCountRepository interface {
	// CountByStage returns a map of stage to count. Stages with no shipments
	// are absent from the map (not present with zero).
	CountByStage(ctx context.Context) (map[Stage]int, error)
}
