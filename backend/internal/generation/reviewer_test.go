package generation

import (
	"context"
	"testing"
)

func TestRunContinuityReviewPasses(t *testing.T) {
	svc := NewService(newFakeStore(), WithReviewer(NewMockReviewer()))

	review, err := svc.RunContinuityReview(context.Background(), "c1", "rev1", "some text")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if review.ReviewType != "CONTINUITY" {
		t.Fatalf("expected CONTINUITY, got %q", review.ReviewType)
	}
	if review.Outcome != "PASS" {
		t.Fatalf("expected PASS, got %q", review.Outcome)
	}
}

func TestRunQualityReviewPasses(t *testing.T) {
	svc := NewService(newFakeStore(), WithReviewer(NewMockReviewer()))

	review, err := svc.RunQualityReview(context.Background(), "c1", "rev1", "some text")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if review.ReviewType != "QUALITY" {
		t.Fatalf("expected QUALITY, got %q", review.ReviewType)
	}
}

func TestRunSafetyReviewPasses(t *testing.T) {
	svc := NewService(newFakeStore(), WithReviewer(NewMockReviewer()))

	review, err := svc.RunSafetyReview(context.Background(), "c1", "rev1", "some text")
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if review.ReviewType != "SAFETY" {
		t.Fatalf("expected SAFETY, got %q", review.ReviewType)
	}
}

func TestRunReviewWithoutReviewerFails(t *testing.T) {
	svc := NewService(newFakeStore())

	if _, err := svc.RunContinuityReview(context.Background(), "c1", "rev1", "text"); err == nil {
		t.Fatal("expected error when reviewer not configured")
	}
}
