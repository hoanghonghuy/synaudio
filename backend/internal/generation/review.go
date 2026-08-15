package generation

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// ChapterReview is a review of a content revision.
type ChapterReview struct {
	ID                string
	ChapterID         string
	ContentRevisionID string
	ReviewType        string
	Outcome           string
	Report            map[string]any
}

// CreateChapterReview records a review for a content revision.
func (s *Service) CreateChapterReview(ctx context.Context, chapterID, contentRevisionID, reviewType, outcome string, report map[string]any) (ChapterReview, error) {
	reviewType = strings.TrimSpace(reviewType)
	if reviewType == "" {
		return ChapterReview{}, errors.New("review type must not be empty")
	}
	if outcome == "" {
		outcome = "PASS"
	}

	r := ChapterReview{
		ID:                uuid.NewString(),
		ChapterID:         chapterID,
		ContentRevisionID: contentRevisionID,
		ReviewType:        reviewType,
		Outcome:           outcome,
		Report:            report,
	}

	return s.store.CreateChapterReview(ctx, r)
}

// ListChapterReviews returns all reviews for a chapter.
func (s *Service) ListChapterReviews(ctx context.Context, chapterID string) ([]ChapterReview, error) {
	return s.store.ListChapterReviews(ctx, chapterID)
}
