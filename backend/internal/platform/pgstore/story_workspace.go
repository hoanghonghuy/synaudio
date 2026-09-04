package pgstore

import (
	"context"

	"github.com/synaudio/synaudio/backend/internal/story"
)

func (s *StoryStore) GetStoryWorkspace(ctx context.Context, storyID string) (story.WorkspaceDetails, error) {
	row, err := s.q.GetStoryWorkspace(ctx, toUUID(storyID))
	if err != nil {
		return story.WorkspaceDetails{}, err
	}
	return story.WorkspaceDetails{
		StoryID:        fromUUID(row.StoryID),
		PlanningMode:   row.PlanningMode,
		PlanningPhase:  row.PlanningPhase,
		PublicRating:   fromText(row.PublicRating),
		PublicWarnings: row.PublicWarnings,
		CoverAssetID:   fromUUID(row.CoverAssetID),
	}, nil
}
