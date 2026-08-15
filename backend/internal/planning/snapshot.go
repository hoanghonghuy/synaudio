package planning

import (
	"context"

	"github.com/google/uuid"
)

// ContextSnapshot is an immutable record of the context used for a generation run.
type ContextSnapshot struct {
	ID                    string
	StoryID               string
	ChapterID             string
	BibleVersionID        string
	EndingPlanVersionID   string
	ArcVersionID          string
	ContentProfileVersionID string
	PromptVersion         string
	WorkflowVersion       string
	Provider              string
	Model                 string
}

// CreateContextSnapshot records an immutable context snapshot.
func (s *Service) CreateContextSnapshot(ctx context.Context, storyID, chapterID, bibleVersionID, endingVersionID, arcVersionID, profileVersionID, promptVersion, workflowVersion, provider, model string) (ContextSnapshot, error) {
	sn := ContextSnapshot{
		ID:                     uuid.NewString(),
		StoryID:                storyID,
		ChapterID:              chapterID,
		BibleVersionID:         bibleVersionID,
		EndingPlanVersionID:    endingVersionID,
		ArcVersionID:           arcVersionID,
		ContentProfileVersionID: profileVersionID,
		PromptVersion:          promptVersion,
		WorkflowVersion:        workflowVersion,
		Provider:               provider,
		Model:                  model,
	}

	return s.store.CreateContextSnapshot(ctx, sn)
}

// ListContextSnapshots returns all snapshots for a story.
func (s *Service) ListContextSnapshots(ctx context.Context, storyID string) ([]ContextSnapshot, error) {
	return s.store.ListContextSnapshots(ctx, storyID)
}
