package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// StoryWorkspaceRow projects lifecycle/public/planning fields needed by the
// admin Story workspace without adding mutation semantics to those fields.
type StoryWorkspaceRow struct {
	StoryID         pgtype.UUID
	PlanningMode    string
	PlanningPhase   string
	PublicRating    pgtype.Text
	PublicWarnings  []string
	CoverAssetID    pgtype.UUID
}

func (q *Queries) GetStoryWorkspace(ctx context.Context, storyID pgtype.UUID) (StoryWorkspaceRow, error) {
	row := q.db.QueryRow(ctx, `
SELECT id, planning_mode, planning_phase, public_rating, public_warnings, cover_asset_id
FROM stories
WHERE id = $1`, storyID)

	var item StoryWorkspaceRow
	err := row.Scan(
		&item.StoryID,
		&item.PlanningMode,
		&item.PlanningPhase,
		&item.PublicRating,
		&item.PublicWarnings,
		&item.CoverAssetID,
	)
	return item, err
}
