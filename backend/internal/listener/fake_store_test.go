package listener

import (
	"context"
	"time"
)

type fakeStore struct {
	favorites map[string]map[string]bool
	progress  map[string]map[string]ListeningProgress
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		favorites: map[string]map[string]bool{},
		progress:  map[string]map[string]ListeningProgress{},
	}
}

func (s *fakeStore) AddFavorite(_ context.Context, userID, storyID string) error {
	if s.favorites[userID] == nil {
		s.favorites[userID] = map[string]bool{}
	}
	s.favorites[userID][storyID] = true
	return nil
}

func (s *fakeStore) RemoveFavorite(_ context.Context, userID, storyID string) error {
	if s.favorites[userID] != nil {
		delete(s.favorites[userID], storyID)
	}
	return nil
}

func (s *fakeStore) IsFavorite(_ context.Context, userID, storyID string) (bool, error) {
	return s.favorites[userID][storyID], nil
}

func (s *fakeStore) ListFavorites(_ context.Context, userID string) ([]Favorite, error) {
	out := []Favorite{}
	for storyID := range s.favorites[userID] {
		out = append(out, Favorite{UserID: userID, StoryID: storyID})
	}
	return out, nil
}

func (s *fakeStore) GetProgress(_ context.Context, userID, chapterID string) (ListeningProgress, error) {
	if s.progress[userID] == nil {
		return ListeningProgress{}, ErrProgressNotFound
	}
	p, ok := s.progress[userID][chapterID]
	if !ok {
		return ListeningProgress{}, ErrProgressNotFound
	}
	return p, nil
}

func (s *fakeStore) SaveProgress(_ context.Context, p ListeningProgress) (ListeningProgress, error) {
	if s.progress[p.UserID] == nil {
		s.progress[p.UserID] = map[string]ListeningProgress{}
	}
	s.progress[p.UserID][p.ChapterID] = p
	return p, nil
}

func (s *fakeStore) MarkCompleted(_ context.Context, userID, chapterID string) (ListeningProgress, error) {
	p, err := s.GetProgress(context.Background(), userID, chapterID)
	if err != nil {
		return ListeningProgress{}, err
	}
	p.CompletedAt = time.Now().Format(time.RFC3339)
	s.progress[userID][chapterID] = p
	return p, nil
}
