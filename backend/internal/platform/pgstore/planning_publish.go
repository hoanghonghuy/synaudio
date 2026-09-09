package pgstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/platform/db"
	"github.com/synaudio/synaudio/backend/internal/planning"
)

const lockChapterForPublishSQL = `
SELECT status
FROM chapters
WHERE id = $1
FOR UPDATE`

// PublishChapterAtomically revalidates publish readiness while holding the same
// chapter-scoped narration advisory lock used for narration allocation and
// latest-narration activation, then persists PUBLISHED in one transaction.
func (s *PlanningStore) PublishChapterAtomically(
	ctx context.Context,
	chapterID string,
	validate planning.ChapterPublishValidator,
) (planning.Chapter, error) {
	if s.beginTx == nil {
		return planning.Chapter{}, errors.New("atomic chapter publish requires transaction-capable database")
	}

	tx, err := s.beginTx.Begin(ctx)
	if err != nil {
		return planning.Chapter{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "narration-version:"+chapterID); err != nil {
		return planning.Chapter{}, fmt.Errorf("lock chapter publish: %w", err)
	}

	var status string
	if err := tx.QueryRow(ctx, lockChapterForPublishSQL, toUUID(chapterID)).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return planning.Chapter{}, planning.ErrChapterNotFound
		}
		return planning.Chapter{}, err
	}
	if status != "READY" {
		return planning.Chapter{}, planningPublishNotReady()
	}

	missing, err := validate(ctx)
	if err != nil {
		return planning.Chapter{}, err
	}
	if len(missing) > 0 {
		return planning.Chapter{}, planningPublishNotReady()
	}

	qtx := s.q.WithTx(tx)
	row, err := qtx.UpdateChapterStatus(ctx, db.UpdateChapterStatusParams{
		ID:     toUUID(chapterID),
		Status: "PUBLISHED",
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return planning.Chapter{}, planning.ErrChapterNotFound
		}
		return planning.Chapter{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return planning.Chapter{}, err
	}
	return toChapter(row), nil
}

func planningPublishNotReady() error {
	return planning.ErrPublishNotReady
}
