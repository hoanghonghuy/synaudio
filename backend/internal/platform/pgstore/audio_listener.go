package pgstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/audio"
	"github.com/synaudio/synaudio/backend/internal/planning"
)

const lockListenerChapterStorySQL = `
SELECT c.status, s.status, s.visibility
FROM chapters c
JOIN stories s ON s.id = c.story_id
WHERE c.id = $1
FOR UPDATE OF c, s`

// IssueListenerEligibleAudioURL returns a presigned listener URL only when the
// chapter/story publication state and active READY latest-narration asset are all
// coherent. Eligibility revalidation, asset selection, and presigning run in one
// transaction with row locks on the chapter and parent story so visibility or
// publish-state revocation cannot interleave with URL issuance.
func (s *AudioStore) IssueListenerEligibleAudioURL(
	ctx context.Context,
	chapterID string,
	_ audio.ListenerEligibilityChecker,
	issue audio.ListenerEligibleAudioIssuer,
) (string, error) {
	if s.beginTx == nil {
		return "", errors.New("atomic listener audio selection requires transaction-capable database")
	}

	tx, err := s.beginTx.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var chapterStatus, storyStatus, storyVisibility string
	if err := tx.QueryRow(ctx, lockListenerChapterStorySQL, toUUID(chapterID)).Scan(
		&chapterStatus, &storyStatus, &storyVisibility,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", audio.ErrListenerAudioNotEligible
		}
		return "", err
	}
	if err := planning.EvaluateListenerEligibility(chapterStatus, storyStatus, storyVisibility); err != nil {
		return "", err
	}

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "narration-version:"+chapterID); err != nil {
		return "", fmt.Errorf("lock chapter listener audio: %w", err)
	}

	row, err := s.q.WithTx(tx).GetListenerEligibleActiveAudio(ctx, toUUID(chapterID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", audio.ErrListenerAudioNotEligible
		}
		return "", err
	}

	url, err := issue(ctx, toAudioAsset(row))
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return url, nil
}
