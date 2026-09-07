package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const getLatestChapterGenerationRun = `
SELECT id, run_type, story_id, chapter_id, status, waiting_reason, workflow_version,
       priority, base_canon_version_id, context_snapshot_id, requested_by,
       idempotency_key, started_at, completed_at, created_at
FROM generation_runs
WHERE chapter_id = $1
  AND run_type = 'CHAPTER_GENERATION'
ORDER BY created_at DESC, id DESC
LIMIT 1
`

// GetLatestChapterGenerationRun returns the newest durable chapter-generation
// workflow intent for one chapter. This read is side-effect free and deliberately
// excludes unrelated run types.
func (q *Queries) GetLatestChapterGenerationRun(ctx context.Context, chapterID pgtype.UUID) (GenerationRun, error) {
	row := q.db.QueryRow(ctx, getLatestChapterGenerationRun, chapterID)
	var i GenerationRun
	err := row.Scan(
		&i.ID,
		&i.RunType,
		&i.StoryID,
		&i.ChapterID,
		&i.Status,
		&i.WaitingReason,
		&i.WorkflowVersion,
		&i.Priority,
		&i.BaseCanonVersionID,
		&i.ContextSnapshotID,
		&i.RequestedBy,
		&i.IdempotencyKey,
		&i.StartedAt,
		&i.CompletedAt,
		&i.CreatedAt,
	)
	return i, err
}
