package pgstore

import (
	"context"

	"github.com/synaudio/synaudio/backend/internal/story"
)

func (s *StoryStore) GetGenerationPolicy(ctx context.Context, storyID string) (story.GenerationPolicy, error) {
	row, err := s.q.GetGenerationPolicy(ctx, toUUID(storyID))
	if err != nil {
		return story.GenerationPolicy{}, err
	}
	return story.GenerationPolicy{
		StoryID:                 fromUUID(row.StoryID),
		MinimumAudioDurationSec: int(row.MinimumAudioDurationSec),
		TargetAudioDurationSec:  int(row.TargetAudioDurationSec),
		ContentOrigin:           row.ContentOrigin,
		Language:                row.Language,
		NarrationLanguage:       row.NarrationLanguage,
		PolicyVersion:           int(row.PolicyVersion),
		CreatedBy:               fromUUID(row.CreatedBy),
	}, nil
}
