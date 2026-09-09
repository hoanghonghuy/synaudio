package pgstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/audio"
)

// IssueListenerEligibleAudioURL returns a presigned listener URL only when the
// active READY asset is sourced from the chapter's latest narration revision.
// Selection and URL issuance run under the same chapter-scoped narration advisory
// lock used for version allocation and latest-narration activation so narration
// commits cannot interleave between validation and presigning.
func (s *AudioStore) IssueListenerEligibleAudioURL(
	ctx context.Context,
	chapterID string,
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
