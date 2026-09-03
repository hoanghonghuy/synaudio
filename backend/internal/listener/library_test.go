package listener

import (
	"context"
	"testing"
)

type libraryTestStore struct {
	progress  []LibraryItem
	favorites []FavoriteStory
}

func (s *libraryTestStore) AddFavorite(context.Context, string, string) error { return nil }
func (s *libraryTestStore) RemoveFavorite(context.Context, string, string) error { return nil }
func (s *libraryTestStore) IsFavorite(context.Context, string, string) (bool, error) { return false, nil }
func (s *libraryTestStore) ListFavorites(context.Context, string) ([]Favorite, error) { return nil, nil }
func (s *libraryTestStore) GetProgress(context.Context, string, string) (ListeningProgress, error) { return ListeningProgress{}, ErrProgressNotFound }
func (s *libraryTestStore) SaveProgress(_ context.Context, p ListeningProgress) (ListeningProgress, error) { return p, nil }
func (s *libraryTestStore) MarkCompleted(context.Context, string, string) (ListeningProgress, error) { return ListeningProgress{}, nil }
func (s *libraryTestStore) ApplyRelistenStatus(context.Context, string, string) (int64, error) { return 0, nil }
func (s *libraryTestStore) ListLibraryProgress(context.Context, string) ([]LibraryItem, error) { return append([]LibraryItem(nil), s.progress...), nil }
func (s *libraryTestStore) ListFavoriteStories(context.Context, string) ([]FavoriteStory, error) { return append([]FavoriteStory(nil), s.favorites...), nil }

func TestGetLibrarySelectsLatestUnfinishedOrRelistenItem(t *testing.T) {
	store := &libraryTestStore{
		progress: []LibraryItem{
			{StoryID: "s1", ChapterID: "c3", ChapterNumber: 3, CompletedAt: "2026-09-03T09:00:00Z", RelistenStatus: "NO_RELISTEN_NEEDED", LastListenedAt: "2026-09-03T09:00:00Z"},
			{StoryID: "s1", ChapterID: "c2", ChapterNumber: 2, PositionMs: 42000, RelistenStatus: "NO_RELISTEN_NEEDED", LastListenedAt: "2026-09-03T08:00:00Z"},
			{StoryID: "s2", ChapterID: "c8", ChapterNumber: 8, CompletedAt: "2026-09-01T08:00:00Z", RelistenStatus: "RELISTEN_REQUIRED", LastListenedAt: "2026-09-01T08:00:00Z"},
		},
		favorites: []FavoriteStory{{StoryID: "s1", Slug: "story-one", Title: "Story One"}},
	}

	library, err := NewService(store).GetLibrary(context.Background(), "u1")
	if err != nil {
		t.Fatalf("get library: %v", err)
	}
	if library.ContinueListening == nil || library.ContinueListening.ChapterID != "c2" {
		t.Fatalf("continue item = %#v, want chapter c2", library.ContinueListening)
	}
	if len(library.Recent) != 3 {
		t.Fatalf("recent count = %d, want 3", len(library.Recent))
	}
	if len(library.Completed) != 2 {
		t.Fatalf("completed count = %d, want 2", len(library.Completed))
	}
	if len(library.Favorites) != 1 || library.Favorites[0].StoryID != "s1" {
		t.Fatalf("favorites = %#v", library.Favorites)
	}
}

func TestGetLibraryOffersCompletedChapterAgainWhenRevisionRequiresRelisten(t *testing.T) {
	store := &libraryTestStore{progress: []LibraryItem{
		{StoryID: "s1", ChapterID: "c1", CompletedAt: "2026-09-03T09:00:00Z", RelistenStatus: "RELISTEN_REQUIRED"},
	}}

	library, err := NewService(store).GetLibrary(context.Background(), "u1")
	if err != nil {
		t.Fatalf("get library: %v", err)
	}
	if library.ContinueListening == nil || library.ContinueListening.ChapterID != "c1" {
		t.Fatalf("expected relisten-required completed chapter to be actionable, got %#v", library.ContinueListening)
	}
}
