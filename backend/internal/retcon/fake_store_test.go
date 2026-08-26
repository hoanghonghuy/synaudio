package retcon

import (
	"context"
)

type fakeStore struct {
	retcons map[string][]RetconRequest
}

func newFakeStore() *fakeStore {
	return &fakeStore{retcons: map[string][]RetconRequest{}}
}

func (s *fakeStore) CreateRetconRequest(_ context.Context, r RetconRequest) (RetconRequest, error) {
	s.retcons[r.StoryID] = append(s.retcons[r.StoryID], r)
	return r, nil
}

func (s *fakeStore) GetRetconRequest(_ context.Context, id string) (RetconRequest, error) {
	for _, rs := range s.retcons {
		for _, r := range rs {
			if r.ID == id {
				return r, nil
			}
		}
	}
	return RetconRequest{}, ErrRetconNotFound
}

func (s *fakeStore) ListRetconRequests(_ context.Context, storyID string) ([]RetconRequest, error) {
	return s.retcons[storyID], nil
}

func (s *fakeStore) UpdateRetconRequest(_ context.Context, r RetconRequest) (RetconRequest, error) {
	for storyID, rs := range s.retcons {
		for i, existing := range rs {
			if existing.ID == r.ID {
				s.retcons[storyID][i] = r
				return r, nil
			}
		}
	}
	return RetconRequest{}, ErrRetconNotFound
}
