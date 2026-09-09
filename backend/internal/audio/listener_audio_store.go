package audio

import (
	"context"
	"errors"
)

// ListenerEligibleAudioIssuer is invoked while the chapter narration advisory lock
// is still held so URL issuance cannot interleave with narration commits.
type ListenerEligibleAudioIssuer func(ctx context.Context, asset AudioAsset) (string, error)

// atomicListenerAudioStore selects listener-playable audio and issues its URL under
// one chapter authority boundary so validation, asset identity, and presigning
// cannot interleave with narration commits.
type atomicListenerAudioStore interface {
	IssueListenerEligibleAudioURL(ctx context.Context, chapterID string, issue ListenerEligibleAudioIssuer) (string, error)
}

func (s *Service) issueValidatedListenerAudioURL(ctx context.Context, chapterID string) (string, error) {
	issue := func(ctx context.Context, asset AudioAsset) (string, error) {
		return s.presignAudioAsset(ctx, asset)
	}

	if atomic, ok := s.store.(atomicListenerAudioStore); ok {
		url, err := atomic.IssueListenerEligibleAudioURL(ctx, chapterID, issue)
		if err != nil {
			if errors.Is(err, ErrListenerAudioNotEligible) {
				return "", err
			}
			if errors.Is(err, ErrAudioAssetNotFound) || errors.Is(err, ErrNarrationNotFound) {
				return "", ErrListenerAudioNotEligible
			}
			return "", err
		}
		return url, nil
	}

	if err := s.AssertActiveAudioMatchesLatestNarration(ctx, chapterID); err != nil {
		return "", err
	}
	asset, err := s.GetActiveAudioAsset(ctx, chapterID)
	if err != nil {
		return "", err
	}
	return issue(ctx, asset)
}

func (s *Service) presignAudioAsset(ctx context.Context, asset AudioAsset) (string, error) {
	if s.presigner == nil {
		return "", errors.New("presigner not configured")
	}
	return s.presigner.PresignedGetObject(ctx, asset.StorageKey, PresignExpiry)
}
