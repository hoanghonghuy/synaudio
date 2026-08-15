package generation

import (
	"context"
	"testing"
)

type reviewFakeStore struct {
	*fakeStore
	reviews map[string][]ChapterReview
}

func newReviewFakeStore() *reviewFakeStore {
	return &reviewFakeStore{
		fakeStore: newFakeStore(),
		reviews:   map[string][]ChapterReview{},
	}
}

func (s *reviewFakeStore) CreateChapterReview(ctx context.Context, r ChapterReview) (ChapterReview, error) {
	s.reviews[r.ChapterID] = append(s.reviews[r.ChapterID], r)
	return r, nil
}

func (s *reviewFakeStore) ListChapterReviews(ctx context.Context, chapterID string) ([]ChapterReview, error) {
	return s.reviews[chapterID], nil
}

func TestCreateChapterReview(t *testing.T) {
	store := newReviewFakeStore()
	svc := NewService(store)

	r, err := svc.CreateChapterReview(context.Background(), "c1", "rev1", "CONTINUITY", "PASS", map[string]any{"note": "ok"})
	if err != nil {
		t.Fatalf("create review: %v", err)
	}
	if r.ReviewType != "CONTINUITY" {
		t.Fatalf("expected CONTINUITY, got %q", r.ReviewType)
	}
	if r.Outcome != "PASS" {
		t.Fatalf("expected PASS, got %q", r.Outcome)
	}
}

func TestCreateChapterReviewRejectsEmptyType(t *testing.T) {
	store := newReviewFakeStore()
	svc := NewService(store)

	if _, err := svc.CreateChapterReview(context.Background(), "c1", "rev1", "  ", "PASS", nil); err == nil {
		t.Fatal("expected error for empty review type")
	}
}

func TestListChapterReviewsReturnsAll(t *testing.T) {
	store := newReviewFakeStore()
	svc := NewService(store)

	_, _ = svc.CreateChapterReview(context.Background(), "c1", "rev1", "CONTINUITY", "PASS", nil)
	_, _ = svc.CreateChapterReview(context.Background(), "c1", "rev1", "QUALITY", "PASS", nil)

	reviews, err := svc.ListChapterReviews(context.Background(), "c1")
	if err != nil {
		t.Fatalf("list reviews: %v", err)
	}
	if len(reviews) != 2 {
		t.Fatalf("expected 2 reviews, got %d", len(reviews))
	}
}
