package audio

import (
	"context"
	"errors"
)

var (
	ErrListenerAudioNotEligible = errors.New("listener audio not eligible")
	ErrListenerAudioGateRequired = errors.New("listener audio gate not configured")
)

// ListenerAudioGate enforces chapter/story listener eligibility before presigning.
type ListenerAudioGate interface {
	CheckListenerAudioEligible(ctx context.Context, chapterID string) error
}

func WithListenerAudioGate(g ListenerAudioGate) Option {
	return func(svc *Service) {
		svc.listenerGate = g
	}
}

// SetListenerAudioGate wires listener eligibility after dependent services are composed.
func (s *Service) SetListenerAudioGate(g ListenerAudioGate) {
	s.listenerGate = g
}

// GetListenerAudioURL returns a presigned download URL only for listener-eligible
// published chapters with an active durable audio asset.
func (s *Service) GetListenerAudioURL(ctx context.Context, chapterID string) (string, error) {
	return s.issueValidatedListenerAudioURL(ctx, chapterID)
}
