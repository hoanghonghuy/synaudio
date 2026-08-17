package generation

import (
	"context"
)

// RecordUsage records token/cost usage for a job attempt.
func (s *Service) RecordUsage(ctx context.Context, attemptID string, usage map[string]any) (JobAttempt, error) {
	return s.store.UpdateJobAttemptUsage(ctx, attemptID, usage)
}

// ListUsageByStory returns all job attempts with recorded usage for a story.
func (s *Service) ListUsageByStory(ctx context.Context, storyID string) ([]JobAttempt, error) {
	return s.store.ListUsageByStory(ctx, storyID)
}
