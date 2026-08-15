package generation

import (
	"context"
	"errors"
)

// Reviewer analyzes a content revision for a specific review dimension.
type Reviewer interface {
	Review(ctx context.Context, in ReviewInput) (ReviewOutput, error)
}

// ReviewInput is the content to review.
type ReviewInput struct {
	Text string
}

// ReviewOutput is the result of a review.
type ReviewOutput struct {
	Outcome string
	Report  map[string]any
}

// MockReviewer always returns PASS for development/testing.
type MockReviewer struct{}

func NewMockReviewer() *MockReviewer {
	return &MockReviewer{}
}

func (MockReviewer) Review(_ context.Context, in ReviewInput) (ReviewOutput, error) {
	return ReviewOutput{
		Outcome: "PASS",
		Report: map[string]any{
			"word_count": len(in.Text),
		},
	}, nil
}

// RunContinuityReview runs a continuity review on a content revision.
func (s *Service) RunContinuityReview(ctx context.Context, chapterID, revisionID, text string) (ChapterReview, error) {
	return s.runReview(ctx, chapterID, revisionID, "CONTINUITY", text)
}

// RunQualityReview runs a writing quality review on a content revision.
func (s *Service) RunQualityReview(ctx context.Context, chapterID, revisionID, text string) (ChapterReview, error) {
	return s.runReview(ctx, chapterID, revisionID, "QUALITY", text)
}

// RunSafetyReview runs a content safety review on a content revision.
func (s *Service) RunSafetyReview(ctx context.Context, chapterID, revisionID, text string) (ChapterReview, error) {
	return s.runReview(ctx, chapterID, revisionID, "SAFETY", text)
}

func (s *Service) runReview(ctx context.Context, chapterID, revisionID, reviewType, text string) (ChapterReview, error) {
	if s.reviewer == nil {
		return ChapterReview{}, errors.New("reviewer not configured")
	}

	out, err := s.reviewer.Review(ctx, ReviewInput{Text: text})
	if err != nil {
		return ChapterReview{}, err
	}

	return s.CreateChapterReview(ctx, chapterID, revisionID, reviewType, out.Outcome, out.Report)
}
