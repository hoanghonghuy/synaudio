package story_test

import (
	"context"

	"github.com/synaudio/synaudio/backend/internal/story"
)

type fakeStore struct {
	stories          map[string]story.Story
	policies         map[string]*story.GenerationPolicy
	slugs            map[string]bool
	genres           []story.Genre
	workflowSettings map[string]story.WorkflowSettings
	contentProfiles  map[string][]story.ContentProfileVersion
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		stories:          map[string]story.Story{},
		policies:         map[string]*story.GenerationPolicy{},
		slugs:            map[string]bool{},
		genres:           []story.Genre{},
		workflowSettings: map[string]story.WorkflowSettings{},
		contentProfiles:  map[string][]story.ContentProfileVersion{},
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

func (s *fakeStore) ListGenres(_ context.Context) ([]story.Genre, error) {
	return s.genres, nil
}

func (s *fakeStore) ListStories(_ context.Context, publicOnly bool) ([]story.Story, error) {
	out := make([]story.Story, 0, len(s.stories))
	for _, st := range s.stories {
		if publicOnly && st.Visibility != story.VisibilityPublic {
			continue
		}
		out = append(out, st)
	}
	return out, nil
}

func (s *fakeStore) GetWorkflowSettings(_ context.Context, storyID string) (story.WorkflowSettings, error) {
	ws, ok := s.workflowSettings[storyID]
	if !ok {
		return story.WorkflowSettings{}, story.ErrStoryNotFound
	}
	return ws, nil
}

func (s *fakeStore) UpdateWorkflowSettings(_ context.Context, ws story.WorkflowSettings) (story.WorkflowSettings, error) {
	s.workflowSettings[ws.StoryID] = ws
	return ws, nil
}

func (s *fakeStore) NextContentProfileVersion(_ context.Context, storyID string) (int, error) {
	versions := s.contentProfiles[storyID]
	return len(versions) + 1, nil
}

func (s *fakeStore) CreateContentProfileVersion(_ context.Context, cp story.ContentProfileVersion) (story.ContentProfileVersion, error) {
	s.contentProfiles[cp.StoryID] = append(s.contentProfiles[cp.StoryID], cp)
	return cp, nil
}

func (s *fakeStore) GetCurrentContentProfile(_ context.Context, storyID string) (story.ContentProfileVersion, error) {
	versions := s.contentProfiles[storyID]
	if len(versions) == 0 {
		return story.ContentProfileVersion{}, story.ErrContentProfileNotFound
	}
	return versions[len(versions)-1], nil
}

func (s *fakeStore) GetStory(_ context.Context, storyID string) (story.Story, error) {
	st, ok := s.stories[storyID]
	if !ok {
		return story.Story{}, story.ErrStoryNotFound
	}
	return st, nil
}

func (s *fakeStore) UpdateStory(_ context.Context, st story.Story) (story.Story, error) {
	s.stories[st.ID] = st
	return st, nil
}
