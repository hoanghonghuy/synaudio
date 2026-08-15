package story_test

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/story"
)

func TestCreateStoryPersistsDraftWithPolicy(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	in := story.CreateStoryInput{
		Title:       "The Long Road",
		Description: "A journey across the continent.",
		CreatedBy:   "user-1",
		Policy: story.GenerationPolicyInput{
			MinimumAudioDurationSec: 1200,
			TargetAudioDurationSec:  1800,
			ContentOrigin:           "ORIGINAL",
			Language:                "en",
			NarrationLanguage:       "en",
		},
	}

	s, err := svc.CreateStory(context.Background(), in)
	if err != nil {
		t.Fatalf("create story: %v", err)
	}

	if s.ID == "" {
		t.Fatal("expected story ID")
	}
	if s.Slug == "" {
		t.Fatal("expected slug")
	}
	if s.Status != story.StatusDraft {
		t.Fatalf("expected DRAFT status, got %q", s.Status)
	}
	if s.Visibility != story.VisibilityPrivate {
		t.Fatalf("expected PRIVATE visibility, got %q", s.Visibility)
	}

	policy := store.policies[s.ID]
	if policy == nil {
		t.Fatal("expected generation policy to be stored")
	}
	if policy.MinimumAudioDurationSec != 1200 {
		t.Fatalf("expected min duration 1200, got %d", policy.MinimumAudioDurationSec)
	}
	if policy.PolicyVersion != 1 {
		t.Fatalf("expected policy version 1, got %d", policy.PolicyVersion)
	}
}

func TestCreateStoryRejectsEmptyTitle(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	_, err := svc.CreateStory(context.Background(), story.CreateStoryInput{
		Title:     "   ",
		CreatedBy: "user-1",
	})
	if !errors.Is(err, story.ErrInvalidTitle) {
		t.Fatalf("expected ErrInvalidTitle, got %v", err)
	}
}

func TestCreateStoryRejectsDuplicateSlug(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	in := story.CreateStoryInput{
		Title:     "Same Title",
		CreatedBy: "user-1",
	}
	if _, err := svc.CreateStory(context.Background(), in); err != nil {
		t.Fatalf("first create: %v", err)
	}

	if _, err := svc.CreateStory(context.Background(), in); !errors.Is(err, story.ErrSlugTaken) {
		t.Fatalf("expected ErrSlugTaken, got %v", err)
	}
}

func TestSlugifyNormalizesTitle(t *testing.T) {
	cases := map[string]string{
		"The Long Road":        "the-long-road",
		"  Hello   World  ":    "hello-world",
		"Đường Dài":            "duong-dai",
		"Story: Part 1!":       "story-part-1",
		"Multiple--Dashes":     "multiple-dashes",
	}

	for in, want := range cases {
		got := story.Slugify(in)
		if got != want {
			t.Fatalf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
