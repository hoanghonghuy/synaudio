package story_test

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/story"
)

type fakeActivationChecker struct {
	missing []string
}

func (f *fakeActivationChecker) CheckActivationReady(_ context.Context, _ string) ([]string, error) {
	return f.missing, nil
}

func TestActivateStoryRejectsWhenPlanningDepsMissing(t *testing.T) {
	store := newFakeStore()
	checker := &fakeActivationChecker{missing: []string{"story_bible", "ending_plan"}}
	svc := story.NewService(store, story.WithActivationChecker(checker))

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Status: story.StatusDraft, PlanningMode: "FINITE"}
	store.policies["s1"] = &story.GenerationPolicy{StoryID: "s1"}
	store.contentProfiles["s1"] = []story.ContentProfileVersion{{ID: "cp1", StoryID: "s1"}}

	_, err := svc.ActivateStory(context.Background(), "s1")
	if !errors.Is(err, story.ErrActivationNotReady) {
		t.Fatalf("expected ErrActivationNotReady, got %v", err)
	}
}

func TestActivateStoryRejectsWhenPolicyMissing(t *testing.T) {
	store := newFakeStore()
	checker := &fakeActivationChecker{missing: nil}
	svc := story.NewService(store, story.WithActivationChecker(checker))

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Status: story.StatusDraft, PlanningMode: "FINITE"}
	store.contentProfiles["s1"] = []story.ContentProfileVersion{{ID: "cp1", StoryID: "s1"}}

	_, err := svc.ActivateStory(context.Background(), "s1")
	if !errors.Is(err, story.ErrActivationNotReady) {
		t.Fatalf("expected ErrActivationNotReady, got %v", err)
	}
}

func TestActivateStoryRejectsWhenContentProfileMissing(t *testing.T) {
	store := newFakeStore()
	checker := &fakeActivationChecker{missing: nil}
	svc := story.NewService(store, story.WithActivationChecker(checker))

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Status: story.StatusDraft, PlanningMode: "FINITE"}
	store.policies["s1"] = &story.GenerationPolicy{StoryID: "s1"}

	_, err := svc.ActivateStory(context.Background(), "s1")
	if !errors.Is(err, story.ErrActivationNotReady) {
		t.Fatalf("expected ErrActivationNotReady, got %v", err)
	}
}

func TestActivateStorySucceedsWhenAllDepsReady(t *testing.T) {
	store := newFakeStore()
	checker := &fakeActivationChecker{missing: nil}
	svc := story.NewService(store, story.WithActivationChecker(checker))

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Status: story.StatusDraft, PlanningMode: "FINITE"}
	store.policies["s1"] = &story.GenerationPolicy{StoryID: "s1"}
	store.contentProfiles["s1"] = []story.ContentProfileVersion{{ID: "cp1", StoryID: "s1"}}

	s, err := svc.ActivateStory(context.Background(), "s1")
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	if s.Status != story.StatusActive {
		t.Fatalf("expected ACTIVE, got %q", s.Status)
	}
}

func TestActivateStoryWithoutCheckerFails(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Status: story.StatusDraft, PlanningMode: "FINITE"}
	store.policies["s1"] = &story.GenerationPolicy{StoryID: "s1"}
	store.contentProfiles["s1"] = []story.ContentProfileVersion{{ID: "cp1", StoryID: "s1"}}

	_, err := svc.ActivateStory(context.Background(), "s1")
	if !errors.Is(err, story.ErrActivationNotReady) {
		t.Fatalf("expected ErrActivationNotReady, got %v", err)
	}
}
