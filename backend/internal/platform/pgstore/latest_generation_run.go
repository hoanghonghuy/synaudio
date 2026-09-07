package pgstore

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/generation"
)

func (s *GenerationStore) GetLatestChapterGenerationRun(ctx context.Context, chapterID string) (generation.GenerationRun, error) {
	row, err := s.q.GetLatestChapterGenerationRun(ctx, toUUID(chapterID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generation.GenerationRun{}, generation.ErrGenerationRunNotFound
		}
		return generation.GenerationRun{}, err
	}
	return toGenerationRun(row), nil
}
