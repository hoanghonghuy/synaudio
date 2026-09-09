package audio

import (
	"context"
	"errors"
)

// ListenerEligibleAudioIssuer is invoked while the chapter listener authority
// boundary is still held so URL issuance cannot interleave with eligibility
// revocation or narration commits.
type ListenerEligibleAudioIssuer func(ctx context.Context, asset AudioAsset) (string, error)

// ListenerEligibilityChecker revalidates chapter/story listener eligibility.
type ListenerEligibilityChecker func(ctx context.Context, chapterID string) error

// atomicListenerAudioStore selects listener-playable audio and issues its URL under
// one chapter authority boundary so eligibility, asset identity, and presigning
// cannot interleave with publication-state or narration commits.
type atomicListenerAudioStore interface {
	IssueListenerEligibleAudioURL(
		ctx context.Context,
		chapterID string,
		check ListenerEligibilityChecker,
		issue ListenerEligibleAudioIssuer,
	) (string, error)
}

func (s *Service) issueValidatedListenerAudioURL(ctx context.Context, chapterID string) (string, error) {
	if s.listenerGate == nil {
		return "", ErrListenerAudioGateRequired
	}

	check := func(ctx context.Context, chapterID string) error {
		return s.listenerGate.CheckListenerAudioEligible(ctx, chapterID)
	}
	issue := func(ctx context.Context, asset AudioAsset) (string, error) {
		return s.presignAudioAsset(ctx, asset)
	}

	if atomic, ok := s.store.(atomicListenerAudioStore); ok {
		url, err := atomic.IssueListenerEligibleAudioURL(ctx, chapterID, check, issue)
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

	if err := check(ctx, chapterID); err != nil {
		return "", err
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
