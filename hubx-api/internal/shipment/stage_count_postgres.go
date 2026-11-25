package shipment

import (
	"context"
	"fmt"
)

// postgresStageCountRepository implements StageCountRepository using PostgreSQL.
type postgresStageCountRepository struct {
	db DB
}

// NewPostgresStageCountRepository creates a PostgreSQL-backed stage count repository.
func NewPostgresStageCountRepository(db DB) StageCountRepository {
	return &postgresStageCountRepository{db: db}
}

// CountByStage returns a map of stage to shipment count.
// Stages with no shipments are absent from the map.
func (r *postgresStageCountRepository) CountByStage(ctx context.Context) (map[Stage]int, error) {
	query := `SELECT stage, COUNT(*) FROM shipments GROUP BY stage`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("count by stage: %w", err)
	}
	defer rows.Close()

	counts := make(map[Stage]int)
	for rows.Next() {
		var stage string
		var count int
		if err := rows.Scan(&stage, &count); err != nil {
			return nil, fmt.Errorf("scan stage count: %w", err)
		}
		counts[Stage(stage)] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stage counts: %w", err)
	}

	return counts, nil
}
