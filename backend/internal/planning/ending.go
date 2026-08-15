package planning

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrEndingNotFound = errors.New("ending plan not found")
)

// EndingPlanVersion is a versioned Ending Plan. Always exists, even for
// OPEN_ENDED stories.
type EndingPlanVersion struct {
	ID              string
	StoryID         string
	VersionNo       int
	Content         map[string]any
	BasedOnVersionID string
	CreatedBy       string
}

// CreateEndingVersion creates a new versioned Ending Plan.
func (s *Service) CreateEndingVersion(ctx context.Context, storyID string, content map[string]any, createdBy string) (EndingPlanVersion, error) {
	if len(content) == 0 {
		return EndingPlanVersion{}, errors.New("ending content must not be empty")
	}

	versionNo, err := s.store.NextEndingVersion(ctx, storyID)
	if err != nil {
		return EndingPlanVersion{}, err
	}

	v := EndingPlanVersion{
		ID:        uuid.NewString(),
		StoryID:   storyID,
		VersionNo: versionNo,
		Content:   content,
		CreatedBy: createdBy,
	}

	return s.store.CreateEndingVersion(ctx, v)
}

// GetCurrentEnding returns the latest Ending Plan version.
func (s *Service) GetCurrentEnding(ctx context.Context, storyID string) (EndingPlanVersion, error) {
	return s.store.GetCurrentEnding(ctx, storyID)
}
