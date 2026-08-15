package story_test

import (
	"context"

	"github.com/synaudio/synaudio/backend/internal/story"
)

type fakeStore struct {
	stories  map[string]story.Story
	policies map[string]*story.GenerationPolicy
	slugs    map[string]bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		stories:  map[string]story.Story{},
		policies: map[string]*story.GenerationPolicy{},
		slugs:    map[string]bool{},
	}
}

func (s *fakeStore) CreateStory(_ context.Context, st story.Story) (story.Story, error) {
	s.stories[st.ID] = st
	s.slugs[st.Slug] = true
	return st, nil
}

func (s *fakeStore) CreateGenerationPolicy(_ context.Context, p story.GenerationPolicy) error {
	cp := p
	s.policies[p.StoryID] = &cp
	return nil
}

func (s *fakeStore) SlugExists(_ context.Context, slug string) (bool, error) {
	return s.slugs[slug], nil
}
