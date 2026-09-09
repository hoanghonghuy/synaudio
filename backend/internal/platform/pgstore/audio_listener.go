package pgstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/audio"
)

// GetListenerEligibleActiveAudio returns the active READY asset only when it is
// sourced from the chapter's latest narration revision. The lookup runs under
// the same chapter-scoped narration advisory lock used for version allocation
// and latest-narration activation so narration commits cannot interleave.
func (s *AudioStore) GetListenerEligibleActiveAudio(ctx context.Context, chapterID string) (audio.AudioAsset, error) {
	if s.beginTx == nil {
		return audio.AudioAsset{}, errors.New("atomic listener audio selection requires transaction-capable database")
	}

	tx, err := s.beginTx.Begin(ctx)
	if err != nil {
		return audio.AudioAsset{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "narration-version:"+chapterID); err != nil {
		return audio.AudioAsset{}, fmt.Errorf("lock chapter listener audio: %w", err)
	}

	row, err := s.q.WithTx(tx).GetListenerEligibleActiveAudio(ctx, toUUID(chapterID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return audio.AudioAsset{}, audio.ErrListenerAudioNotEligible
		}
		return audio.AudioAsset{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return audio.AudioAsset{}, err
	}
	return toAudioAsset(row), nil
}
