package story_test

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/story"
)

func TestArchiveStorySavesPreviousStatus(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Status: story.StatusActive}

	s, err := svc.ArchiveStory(context.Background(), "s1")
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	if s.Status != story.StatusArchived {
		t.Fatalf("expected ARCHIVED, got %q", s.Status)
	}
	if s.StatusBeforeArchive != story.StatusActive {
		t.Fatalf("expected status_before_archive ACTIVE, got %q", s.StatusBeforeArchive)
	}
}

func TestRestoreStoryReturnsToPreviousStatus(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.stories["s1"] = story.Story{
		ID: "s1", Slug: "a", Title: "A",
		Status: story.StatusArchived, StatusBeforeArchive: story.StatusCompleted,
	}

	s, err := svc.RestoreStory(context.Background(), "s1")
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if s.Status != story.StatusCompleted {
		t.Fatalf("expected COMPLETED, got %q", s.Status)
	}
}

func TestMakePublicRequiresActiveOrCompleted(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Status: story.StatusDraft}

	_, err := svc.MakePublic(context.Background(), "s1")
	if !errors.Is(err, story.ErrNotPublicable) {
		t.Fatalf("expected ErrNotPublicable, got %v", err)
	}
}

func TestMakePublicSucceedsForActive(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Status: story.StatusActive}

	s, err := svc.MakePublic(context.Background(), "s1")
	if err != nil {
		t.Fatalf("make public: %v", err)
	}
	if s.Visibility != story.VisibilityPublic {
		t.Fatalf("expected PUBLIC, got %q", s.Visibility)
	}
}

func TestMakePrivateSetsPrivate(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Visibility: story.VisibilityPublic}

	s, err := svc.MakePrivate(context.Background(), "s1")
	if err != nil {
		t.Fatalf("make private: %v", err)
	}
	if s.Visibility != story.VisibilityPrivate {
		t.Fatalf("expected PRIVATE, got %q", s.Visibility)
	}
}
