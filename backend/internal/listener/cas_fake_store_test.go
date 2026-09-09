package listener

import (
	"context"
	"sync"
	"time"
)

type casFakeStore struct {
	mu       sync.Mutex
	progress map[string]map[string]ListeningProgress
}

func newCASFakeStore() *casFakeStore {
	return &casFakeStore{
		progress: map[string]map[string]ListeningProgress{},
	}
}

func (s *casFakeStore) AddFavorite(_ context.Context, userID, storyID string) error {
	return nil
}

func (s *casFakeStore) RemoveFavorite(_ context.Context, userID, storyID string) error {
	return nil
}

func (s *casFakeStore) IsFavorite(_ context.Context, userID, storyID string) (bool, error) {
	return false, nil
}

func (s *casFakeStore) ListFavorites(_ context.Context, userID string) ([]Favorite, error) {
	return nil, nil
}

func (s *casFakeStore) GetProgress(_ context.Context, userID, chapterID string) (ListeningProgress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.progress[userID] == nil {
		return ListeningProgress{}, ErrProgressNotFound
	}
	p, ok := s.progress[userID][chapterID]
	if !ok {
		return ListeningProgress{}, ErrProgressNotFound
	}
	return p, nil
}

func (s *casFakeStore) SaveProgress(_ context.Context, p ListeningProgress, expectedVersion int64) (ListeningProgress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentVersion := int64(0)
	var current ListeningProgress
	if s.progress[p.UserID] != nil {
		if existing, ok := s.progress[p.UserID][p.ChapterID]; ok {
			current = existing
			currentVersion = existing.Version
		}
	}

	if currentVersion != expectedVersion {
		if currentVersion == 0 {
			return ListeningProgress{}, ErrProgressNotFound
		}
		return ListeningProgress{}, &ProgressVersionConflict{Current: current}
	}

	p.Version = expectedVersion + 1
	if s.progress[p.UserID] == nil {
		s.progress[p.UserID] = map[string]ListeningProgress{}
	}
	s.progress[p.UserID][p.ChapterID] = p
	return p, nil
}

func (s *casFakeStore) MarkCompleted(_ context.Context, userID, chapterID string) (ListeningProgress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.progress[userID] == nil {
		return ListeningProgress{}, ErrProgressNotFound
	}
	p, ok := s.progress[userID][chapterID]
	if !ok {
		return ListeningProgress{}, ErrProgressNotFound
	}
	p.CompletedAt = time.Now().Format(time.RFC3339)
	s.progress[userID][chapterID] = p
	return p, nil
}

func (s *casFakeStore) ApplyRelistenStatus(_ context.Context, chapterID, status string) (int64, error) {
	return 0, nil
}
