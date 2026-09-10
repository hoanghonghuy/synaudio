package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const getActiveOfficialCanonBranch = `
SELECT id, story_id, type, status, base_version_id, generation_run_id, retcon_request_id, created_at
FROM canon_branches
WHERE story_id = $1
  AND type = 'OFFICIAL'
  AND status = 'ACTIVE'
ORDER BY created_at DESC, id DESC
LIMIT 1
`

// GetActiveOfficialCanonBranch returns the authoritative active OFFICIAL canon branch for a story.
// This query is kept as a small db extension until the surrounding planning contract is regenerated.
func (q *Queries) GetActiveOfficialCanonBranch(ctx context.Context, storyID pgtype.UUID) (CanonBranch, error) {
	row := q.db.QueryRow(ctx, getActiveOfficialCanonBranch, storyID)
	var i CanonBranch
	err := row.Scan(
		&i.ID,
		&i.StoryID,
		&i.Type,
		&i.Status,
		&i.BaseVersionID,
		&i.GenerationRunID,
		&i.RetconRequestID,
		&i.CreatedAt,
	)
	return i, err
}
