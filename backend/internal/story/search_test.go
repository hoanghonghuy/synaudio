package story_test

import (
	"context"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/story"
)

func TestSearchStoriesFiltersByGenre(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Visibility: story.VisibilityPublic}
	store.stories["s2"] = story.Story{ID: "s2", Slug: "b", Title: "B", Visibility: story.VisibilityPublic}
	store.storyGenres["s1"] = []string{"fantasy"}
	store.storyGenres["s2"] = []string{"romance"}

	stories, err := svc.SearchStories(context.Background(), story.SearchStoriesInput{
		Genre: "fantasy",
	})
	if err != nil {
		t.Fatalf("search stories: %v", err)
	}
	if len(stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(stories))
	}
	if stories[0].ID != "s1" {
		t.Fatalf("expected s1, got %s", stories[0].ID)
	}
}

func TestSearchStoriesOnlyReturnsPublic(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Visibility: story.VisibilityPublic}
	store.stories["s2"] = story.Story{ID: "s2", Slug: "b", Title: "B", Visibility: story.VisibilityPrivate}

	stories, err := svc.SearchStories(context.Background(), story.SearchStoriesInput{})
	if err != nil {
		t.Fatalf("search stories: %v", err)
	}
	if len(stories) != 1 {
		t.Fatalf("expected 1 public story, got %d", len(stories))
	}
	if stories[0].ID != "s1" {
		t.Fatalf("expected s1, got %s", stories[0].ID)
	}
}

func TestSearchStoriesPassesQueryAndSort(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Visibility: story.VisibilityPublic}

	_, err := svc.SearchStories(context.Background(), story.SearchStoriesInput{
		Query: "road",
		Sort:  story.SortNew,
	})
	if err != nil {
		t.Fatalf("search stories: %v", err)
	}

	if store.lastSearchQuery != "road" {
		t.Fatalf("expected query 'road', got %q", store.lastSearchQuery)
	}
	if store.lastSearchSort != story.SortNew {
		t.Fatalf("expected sort NEW, got %q", store.lastSearchSort)
	}
}
