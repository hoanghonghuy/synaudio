package story_test

import (
	"context"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/story"
)

func TestListGenresReturnsAll(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.genres = []story.Genre{
		{ID: "g1", Slug: "fantasy", Name: "Fantasy"},
		{ID: "g2", Slug: "romance", Name: "Romance"},
	}

	genres, err := svc.ListGenres(context.Background())
	if err != nil {
		t.Fatalf("list genres: %v", err)
	}
	if len(genres) != 2 {
		t.Fatalf("expected 2 genres, got %d", len(genres))
	}
	if genres[0].Slug != "fantasy" {
		t.Fatalf("expected first genre fantasy, got %q", genres[0].Slug)
	}
}
