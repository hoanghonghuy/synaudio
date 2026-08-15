package story_test

import (
	"context"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/story"
)

func TestListStoriesPublicOnlyReturnsPublic(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Visibility: story.VisibilityPublic}
	store.stories["s2"] = story.Story{ID: "s2", Slug: "b", Title: "B", Visibility: story.VisibilityPrivate}

	stories, err := svc.ListStories(context.Background(), story.ListStoriesInput{
		PublicOnly: true,
	})
	if err != nil {
		t.Fatalf("list stories: %v", err)
	}
	if len(stories) != 1 {
		t.Fatalf("expected 1 public story, got %d", len(stories))
	}
	if stories[0].ID != "s1" {
		t.Fatalf("expected s1, got %s", stories[0].ID)
	}
}

func TestListStoriesAdminReturnsAll(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Visibility: story.VisibilityPublic}
	store.stories["s2"] = story.Story{ID: "s2", Slug: "b", Title: "B", Visibility: story.VisibilityPrivate}

	stories, err := svc.ListStories(context.Background(), story.ListStoriesInput{
		PublicOnly: false,
	})
	if err != nil {
		t.Fatalf("list stories: %v", err)
	}
	if len(stories) != 2 {
		t.Fatalf("expected 2 stories, got %d", len(stories))
	}
}
