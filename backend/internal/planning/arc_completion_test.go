package planning

import (
	"context"
	"testing"
)

func TestReviewArcCompletionAllPublished(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	store.arcs["s1"] = []StoryArc{{ID: "a1", StoryID: "s1", Ordinal: 1, Status: "ACTIVE"}}
	store.chapters["s1"] = []Chapter{
		{ID: "ch1", StoryID: "s1", ChapterNumber: 1, Status: "PUBLISHED", ArcID: "a1"},
		{ID: "ch2", StoryID: "s1", ChapterNumber: 2, Status: "PUBLISHED", ArcID: "a1"},
	}

	res, err := svc.ReviewArcCompletion(context.Background(), "s1", "a1")
	if err != nil {
		t.Fatalf("review arc: %v", err)
	}
	if !res.Complete {
		t.Fatal("expected arc complete")
	}
	if res.TotalChapters != 2 {
		t.Fatalf("expected 2 chapters, got %d", res.TotalChapters)
	}
	if res.CompletedChapters != 2 {
		t.Fatalf("expected 2 completed, got %d", res.CompletedChapters)
	}
}

func TestReviewArcCompletionIncomplete(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	store.arcs["s1"] = []StoryArc{{ID: "a1", StoryID: "s1", Ordinal: 1, Status: "ACTIVE"}}
	store.chapters["s1"] = []Chapter{
		{ID: "ch1", StoryID: "s1", ChapterNumber: 1, Status: "PUBLISHED", ArcID: "a1"},
		{ID: "ch2", StoryID: "s1", ChapterNumber: 2, Status: "DRAFT", ArcID: "a1"},
	}

	res, err := svc.ReviewArcCompletion(context.Background(), "s1", "a1")
	if err != nil {
		t.Fatalf("review arc: %v", err)
	}
	if res.Complete {
		t.Fatal("expected arc incomplete")
	}
	if res.CompletedChapters != 1 {
		t.Fatalf("expected 1 completed, got %d", res.CompletedChapters)
	}
}

func TestReviewArcCompletionArcNotFound(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.ReviewArcCompletion(context.Background(), "s1", "missing"); err == nil {
		t.Fatal("expected error for missing arc")
	}
}
