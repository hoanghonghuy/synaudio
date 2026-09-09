package planning

import (
	"context"
	"errors"
)

var ErrListenerAudioNotEligible = errors.New("listener audio not eligible")

// StoryVisibilityReader exposes the listener-facing story publication state.
type StoryVisibilityReader interface {
	GetStoryVisibility(ctx context.Context, storyID string) (status, visibility string, err error)
}

// ListenerEligibility enforces chapter/story listener eligibility for public audio.
type ListenerEligibility struct {
	chapters Store
	stories  StoryVisibilityReader
}

func NewListenerEligibility(chapters Store, stories StoryVisibilityReader) *ListenerEligibility {
	return &ListenerEligibility{chapters: chapters, stories: stories}
}

// CheckListenerAudioEligible returns nil only when the chapter is published and
// its parent story is listener-visible.
func (l *ListenerEligibility) CheckListenerAudioEligible(ctx context.Context, chapterID string) error {
	if l == nil || l.chapters == nil || l.stories == nil {
		return ErrListenerAudioNotEligible
	}

	ch, err := l.chapters.GetChapter(ctx, chapterID)
	if err != nil {
		return err
	}
	if ch.Status != "PUBLISHED" {
		return ErrListenerAudioNotEligible
	}

	status, visibility, err := l.stories.GetStoryVisibility(ctx, ch.StoryID)
	if err != nil {
		return err
	}
	if visibility != "PUBLIC" {
		return ErrListenerAudioNotEligible
	}
	if status != "ACTIVE" && status != "COMPLETED" {
		return ErrListenerAudioNotEligible
	}

	return nil
}
