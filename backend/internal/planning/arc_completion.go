package planning

import (
	"context"
)

// ArcCompletionResult is the outcome of an arc completion review.
type ArcCompletionResult struct {
	ArcID             string
	Complete          bool
	TotalChapters     int
	CompletedChapters int
	PendingChapters   []string
}

// ReviewArcCompletion checks whether all chapters in an arc are PUBLISHED.
// This is advisory: it surfaces whether an arc is complete, but does not
// auto-transition any state.
func (s *Service) ReviewArcCompletion(ctx context.Context, storyID, arcID string) (ArcCompletionResult, error) {
	if _, err := s.store.GetArc(ctx, arcID); err != nil {
		return ArcCompletionResult{}, err
	}

	chapters, err := s.store.ListChapters(ctx, storyID)
	if err != nil {
		return ArcCompletionResult{}, err
	}

	res := ArcCompletionResult{ArcID: arcID}
	for _, c := range chapters {
		if c.ArcID != arcID {
			continue
		}
		res.TotalChapters++
		if c.Status == "PUBLISHED" {
			res.CompletedChapters++
		} else {
			res.PendingChapters = append(res.PendingChapters, c.ID)
		}
	}

	res.Complete = res.TotalChapters > 0 && res.CompletedChapters == res.TotalChapters
	return res, nil
}
