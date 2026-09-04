package story

import (
	"context"
	"errors"
)

// WorkspaceDetails contains Story planning/public fields needed by Admin UI.
// It is a read projection; mutation validity remains in dedicated domain APIs.
type WorkspaceDetails struct {
	StoryID        string
	PlanningMode   string
	PlanningPhase  string
	PublicRating   string
	PublicWarnings []string
	CoverAssetID   string
}

type workspaceReader interface {
	GetStoryWorkspace(ctx context.Context, storyID string) (WorkspaceDetails, error)
}

func (s *Service) GetWorkspaceDetails(ctx context.Context, storyID string) (WorkspaceDetails, error) {
	reader, ok := s.store.(workspaceReader)
	if !ok {
		return WorkspaceDetails{}, errors.New("story workspace reader unavailable")
	}
	return reader.GetStoryWorkspace(ctx, storyID)
}
