package listener

import (
	"context"
	"errors"
)

var (
	ErrProgressNotFound         = errors.New("listening progress not found")
	ErrProgressVersionConflict  = errors.New("progress version conflict")
)

// ProgressVersionConflict reports an optimistic concurrency failure and carries
// the authoritative server progress for client refresh.
type ProgressVersionConflict struct {
	Current ListeningProgress
}

func (e *ProgressVersionConflict) Error() string {
	return ErrProgressVersionConflict.Error()
}

func (e *ProgressVersionConflict) Unwrap() error {
	return ErrProgressVersionConflict
}

// Favorite is a user's favorited story.
type Favorite struct {
	UserID  string
	StoryID string
}

// ListeningProgress tracks a user's playback position in a chapter.
type ListeningProgress struct {
	UserID               string
	ChapterID            string
	PositionMs           int64
	CompletedAt          string
	LastAudioAssetID     string
	LastPlaybackSessionID string
	Version              int64
	RelistenStatus       string
}

// Store is the persistence boundary for the listener service.
type Store interface {
	AddFavorite(ctx context.Context, userID, storyID string) error
	RemoveFavorite(ctx context.Context, userID, storyID string) error
	IsFavorite(ctx context.Context, userID, storyID string) (bool, error)
	ListFavorites(ctx context.Context, userID string) ([]Favorite, error)

	GetProgress(ctx context.Context, userID, chapterID string) (ListeningProgress, error)
	SaveProgress(ctx context.Context, p ListeningProgress, expectedVersion int64) (ListeningProgress, error)
	MarkCompleted(ctx context.Context, userID, chapterID string) (ListeningProgress, error)

	ApplyRelistenStatus(ctx context.Context, chapterID, status string) (int64, error)
}

// Service orchestrates listener-facing features.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// AddFavorite favorites a story (idempotent).
func (s *Service) AddFavorite(ctx context.Context, userID, storyID string) error {
	return s.store.AddFavorite(ctx, userID, storyID)
}

// RemoveFavorite unfavorites a story.
func (s *Service) RemoveFavorite(ctx context.Context, userID, storyID string) error {
	return s.store.RemoveFavorite(ctx, userID, storyID)
}

// IsFavorite reports whether a user favorited a story.
func (s *Service) IsFavorite(ctx context.Context, userID, storyID string) (bool, error) {
	return s.store.IsFavorite(ctx, userID, storyID)
}

// ListFavorites returns all favorites for a user.
func (s *Service) ListFavorites(ctx context.Context, userID string) ([]Favorite, error) {
	return s.store.ListFavorites(ctx, userID)
}

// SaveProgress records a playback position update using optimistic concurrency on
// expectedVersion. Writers that observed version N must pass N; the store commits
// version N+1 only when the server row is still at N.
func (s *Service) SaveProgress(ctx context.Context, userID, chapterID string, positionMs int64, audioAssetID, sessionID string, expectedVersion int64) (ListeningProgress, error) {
	p := ListeningProgress{
		UserID:                userID,
		ChapterID:             chapterID,
		PositionMs:            positionMs,
		LastAudioAssetID:      audioAssetID,
		LastPlaybackSessionID: sessionID,
	}

	return s.store.SaveProgress(ctx, p, expectedVersion)
}

// GetProgress returns a user's progress for a chapter.
func (s *Service) GetProgress(ctx context.Context, userID, chapterID string) (ListeningProgress, error) {
	return s.store.GetProgress(ctx, userID, chapterID)
}

// MarkCompleted marks a chapter as completed for a user.
func (s *Service) MarkCompleted(ctx context.Context, userID, chapterID string) (ListeningProgress, error) {
	return s.store.MarkCompleted(ctx, userID, chapterID)
}

// ApplyRevisionImpact marks all listeners of a chapter with a relisten status
// after a retcon changes the published content. Completion is preserved.
func (s *Service) ApplyRevisionImpact(ctx context.Context, chapterID, relistenStatus string) (int64, error) {
	return s.store.ApplyRelistenStatus(ctx, chapterID, relistenStatus)
}
