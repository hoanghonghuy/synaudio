package listener

import (
	"context"
	"errors"
)

var ErrLibraryNotConfigured = errors.New("listener library projection not configured")

// LibraryItem is the server-backed resume/read model for one listened chapter.
// It deliberately carries the chapter's current durable audio pointer rather
// than treating the last-listened storage locator as canonical after a revision.
type LibraryItem struct {
	StoryID          string `json:"story_id"`
	StorySlug        string `json:"story_slug"`
	StoryTitle       string `json:"story_title"`
	StoryDescription string `json:"story_description"`
	ChapterID        string `json:"chapter_id"`
	ChapterNumber    int32  `json:"chapter_number"`
	ChapterTitle     string `json:"chapter_title"`
	PositionMs       int64  `json:"position_ms"`
	CompletedAt      string `json:"completed_at,omitempty"`
	RelistenStatus   string `json:"relisten_status"`
	LastListenedAt   string `json:"last_listened_at,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
	AudioAssetID     string `json:"audio_asset_id,omitempty"`
	AudioDurationMs  int32  `json:"audio_duration_ms,omitempty"`
}

// FavoriteStory is a useful favorite projection rather than an opaque story ID.
type FavoriteStory struct {
	StoryID          string `json:"story_id"`
	Slug             string `json:"slug"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	FavoritedAt      string `json:"favorited_at"`
}

// Library is the authenticated listener dashboard projection. ContinueListening
// is at most one latest unfinished/relisten-required entry; Recent and Completed
// remain deterministically ordered by server timestamps.
type Library struct {
	ContinueListening *LibraryItem    `json:"continue_listening,omitempty"`
	Recent            []LibraryItem   `json:"recent"`
	Completed         []LibraryItem   `json:"completed"`
	Favorites         []FavoriteStory `json:"favorites"`
}

// LibraryStore is an additive read-model capability so existing listener stores
// and focused unit fakes are not forced to implement dashboard queries.
type LibraryStore interface {
	ListLibraryProgress(ctx context.Context, userID string) ([]LibraryItem, error)
	ListFavoriteStories(ctx context.Context, userID string) ([]FavoriteStory, error)
}

func (s *Service) GetLibrary(ctx context.Context, userID string) (Library, error) {
	store, ok := s.store.(LibraryStore)
	if !ok {
		return Library{}, ErrLibraryNotConfigured
	}
	progress, err := store.ListLibraryProgress(ctx, userID)
	if err != nil {
		return Library{}, err
	}
	favorites, err := store.ListFavoriteStories(ctx, userID)
	if err != nil {
		return Library{}, err
	}

	result := Library{
		Recent:    make([]LibraryItem, 0, len(progress)),
		Completed: make([]LibraryItem, 0),
		Favorites: favorites,
	}
	for _, item := range progress {
		result.Recent = append(result.Recent, item)
		if item.CompletedAt != "" {
			result.Completed = append(result.Completed, item)
		}
		if result.ContinueListening == nil && (item.CompletedAt == "" || item.RelistenStatus != "NO_RELISTEN_NEEDED") {
			copy := item
			result.ContinueListening = &copy
		}
	}
	return result, nil
}
