package pgstore

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/listener"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

// ListenerStore implements listener.Store backed by PostgreSQL via sqlc.
type ListenerStore struct {
	q *db.Queries
}

func NewListenerStore(q *db.Queries) *ListenerStore {
	return &ListenerStore{q: q}
}

// ============================================================
// Favorites
// ============================================================

func (s *ListenerStore) AddFavorite(ctx context.Context, userID, storyID string) error {
	return s.q.AddFavorite(ctx, db.AddFavoriteParams{
		UserID:  toUUID(userID),
		StoryID: toUUID(storyID),
	})
}

func (s *ListenerStore) RemoveFavorite(ctx context.Context, userID, storyID string) error {
	return s.q.RemoveFavorite(ctx, db.RemoveFavoriteParams{
		UserID:  toUUID(userID),
		StoryID: toUUID(storyID),
	})
}

func (s *ListenerStore) IsFavorite(ctx context.Context, userID, storyID string) (bool, error) {
	return s.q.IsFavorite(ctx, db.IsFavoriteParams{
		UserID:  toUUID(userID),
		StoryID: toUUID(storyID),
	})
}

func (s *ListenerStore) ListFavorites(ctx context.Context, userID string) ([]listener.Favorite, error) {
	rows, err := s.q.ListFavorites(ctx, toUUID(userID))
	if err != nil {
		return nil, err
	}
	out := make([]listener.Favorite, 0, len(rows))
	for _, r := range rows {
		out = append(out, listener.Favorite{
			UserID:  fromUUID(r.UserID),
			StoryID: fromUUID(r.StoryID),
		})
	}
	return out, nil
}

// ============================================================
// Listening Progress
// ============================================================

func (s *ListenerStore) GetProgress(ctx context.Context, userID, chapterID string) (listener.ListeningProgress, error) {
	row, err := s.q.GetProgress(ctx, db.GetProgressParams{
		UserID:    toUUID(userID),
		ChapterID: toUUID(chapterID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return listener.ListeningProgress{}, listener.ErrProgressNotFound
		}
		return listener.ListeningProgress{}, err
	}
	return listener.ListeningProgress{
		UserID:                fromUUID(row.UserID),
		ChapterID:             fromUUID(row.ChapterID),
		PositionMs:            row.PositionMs,
		CompletedAt:           fromTimestamptz(row.CompletedAt),
		LastAudioAssetID:      fromUUID(row.LastAudioAssetID),
		LastPlaybackSessionID: fromUUID(row.LastPlaybackSessionID),
		Version:               row.Version,
		RelistenStatus:        row.RelistenStatus,
	}, nil
}

func (s *ListenerStore) SaveProgress(ctx context.Context, p listener.ListeningProgress) (listener.ListeningProgress, error) {
	row, err := s.q.SaveProgress(ctx, db.SaveProgressParams{
		UserID:                toUUID(p.UserID),
		ChapterID:             toUUID(p.ChapterID),
		PositionMs:            p.PositionMs,
		LastAudioAssetID:      toUUID(p.LastAudioAssetID),
		LastPlaybackSessionID: toUUID(p.LastPlaybackSessionID),
		Version:               p.Version,
	})
	if err != nil {
		return listener.ListeningProgress{}, err
	}
	return listener.ListeningProgress{
		UserID:                fromUUID(row.UserID),
		ChapterID:             fromUUID(row.ChapterID),
		PositionMs:            row.PositionMs,
		CompletedAt:           fromTimestamptz(row.CompletedAt),
		LastAudioAssetID:      fromUUID(row.LastAudioAssetID),
		LastPlaybackSessionID: fromUUID(row.LastPlaybackSessionID),
		Version:               row.Version,
		RelistenStatus:        row.RelistenStatus,
	}, nil
}

func (s *ListenerStore) MarkCompleted(ctx context.Context, userID, chapterID string) (listener.ListeningProgress, error) {
	row, err := s.q.MarkCompleted(ctx, db.MarkCompletedParams{
		UserID:    toUUID(userID),
		ChapterID: toUUID(chapterID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return listener.ListeningProgress{}, listener.ErrProgressNotFound
		}
		return listener.ListeningProgress{}, err
	}
	return listener.ListeningProgress{
		UserID:                fromUUID(row.UserID),
		ChapterID:             fromUUID(row.ChapterID),
		PositionMs:            row.PositionMs,
		CompletedAt:           fromTimestamptz(row.CompletedAt),
		LastAudioAssetID:      fromUUID(row.LastAudioAssetID),
		LastPlaybackSessionID: fromUUID(row.LastPlaybackSessionID),
		Version:               row.Version,
		RelistenStatus:        row.RelistenStatus,
	}, nil
}

func (s *ListenerStore) ApplyRelistenStatus(ctx context.Context, chapterID, status string) (int64, error) {
	rows, err := s.q.ApplyRelistenStatus(ctx, db.ApplyRelistenStatusParams{
		ChapterID:     toUUID(chapterID),
		RelistenStatus: status,
	})
	if err != nil {
		return 0, err
	}
	return int64(len(rows)), nil
}

// ============================================================
// Converters
// ============================================================

