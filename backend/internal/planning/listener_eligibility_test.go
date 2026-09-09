package planning

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/audio"
)

type fakeStoryVisibilityReader struct {
	status     string
	visibility string
	err        error
}

func (f *fakeStoryVisibilityReader) GetStoryVisibility(ctx context.Context, storyID string) (string, string, error) {
	return f.status, f.visibility, f.err
}

func TestListenerEligibilityRejectsUnpublishedChapter(t *testing.T) {
	store := newPublishFakeStore()
	ch, _ := store.CreateChapter(context.Background(), Chapter{ID: "ch-1", StoryID: "s1", Status: "READY"})
	gate := NewListenerEligibility(store, &fakeStoryVisibilityReader{status: "ACTIVE", visibility: "PUBLIC"})

	if err := gate.CheckListenerAudioEligible(context.Background(), ch.ID); !errors.Is(err, audio.ErrListenerAudioNotEligible) {
		t.Fatalf("expected ErrListenerAudioNotEligible, got %v", err)
	}
}

func TestListenerEligibilityRejectsPrivateStory(t *testing.T) {
	store := newPublishFakeStore()
	ch, _ := store.CreateChapter(context.Background(), Chapter{ID: "ch-1", StoryID: "s1", Status: "PUBLISHED"})
	gate := NewListenerEligibility(store, &fakeStoryVisibilityReader{status: "ACTIVE", visibility: "PRIVATE"})

	if err := gate.CheckListenerAudioEligible(context.Background(), ch.ID); !errors.Is(err, audio.ErrListenerAudioNotEligible) {
		t.Fatalf("expected ErrListenerAudioNotEligible, got %v", err)
	}
}

func TestListenerEligibilityAllowsPublishedPublicStoryChapter(t *testing.T) {
	store := newPublishFakeStore()
	ch, _ := store.CreateChapter(context.Background(), Chapter{ID: "ch-1", StoryID: "s1", Status: "PUBLISHED"})
	gate := NewListenerEligibility(store, &fakeStoryVisibilityReader{status: "ACTIVE", visibility: "PUBLIC"})

	if err := gate.CheckListenerAudioEligible(context.Background(), ch.ID); err != nil {
		t.Fatalf("expected eligible chapter, got %v", err)
	}
}
