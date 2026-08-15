package planning

import (
	"context"
)

// CheckActivationReady reports which planning foundation dependencies are
// missing for a story, satisfying the story.ActivationChecker contract.
func (s *Service) CheckActivationReady(ctx context.Context, storyID string) ([]string, error) {
	missing := []string{}

	if _, err := s.store.GetCurrentBible(ctx, storyID); err != nil {
		missing = append(missing, "story_bible")
	}

	if _, err := s.store.GetCurrentEnding(ctx, storyID); err != nil {
		missing = append(missing, "ending_plan")
	}

	arcs, err := s.store.ListArcs(ctx, storyID)
	if err != nil {
		return nil, err
	}
	if len(arcs) == 0 {
		missing = append(missing, "arc")
	}

	characters, err := s.store.ListCharacters(ctx, storyID)
	if err != nil {
		return nil, err
	}
	if len(characters) == 0 {
		missing = append(missing, "character")
	}

	return missing, nil
}
