package story_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/story"
)

func TestCheckActivationReadinessReturnsActionableMissingDependencies(t *testing.T) {
	store := newFakeStore()
	checker := &fakeActivationChecker{missing: []string{"story_bible", "ending_plan", "initial_arc", "main_character"}}
	svc := story.NewService(store, story.WithActivationChecker(checker))

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Status: story.StatusDraft}

	got, err := svc.CheckActivationReadiness(context.Background(), "s1")
	if err != nil {
		t.Fatalf("check readiness: %v", err)
	}
	want := []string{
		"planning_mode",
		"generation_policy",
		"content_profile",
		"story_bible",
		"ending_plan",
		"initial_arc",
		"main_character",
	}
	if got.Ready {
		t.Fatal("expected readiness to be false")
	}
	if !reflect.DeepEqual(got.Missing, want) {
		t.Fatalf("missing prerequisites = %#v, want %#v", got.Missing, want)
	}
}

func TestCheckActivationReadinessReadyWhenAllDependenciesExist(t *testing.T) {
	store := newFakeStore()
	checker := &fakeActivationChecker{}
	svc := story.NewService(store, story.WithActivationChecker(checker))

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Status: story.StatusDraft, PlanningMode: "FINITE"}
	store.policies["s1"] = &story.GenerationPolicy{StoryID: "s1"}
	store.contentProfiles["s1"] = []story.ContentProfileVersion{{ID: "cp1", StoryID: "s1"}}

	got, err := svc.CheckActivationReadiness(context.Background(), "s1")
	if err != nil {
		t.Fatalf("check readiness: %v", err)
	}
	if !got.Ready {
		t.Fatalf("expected ready, missing %#v", got.Missing)
	}
	if len(got.Missing) != 0 {
		t.Fatalf("expected no missing prerequisites, got %#v", got.Missing)
	}
}
