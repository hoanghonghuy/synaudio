package audio

import (
	"context"
	"errors"
)

// atomicListenerAudioStore selects listener-playable audio under one chapter
// authority boundary so validation and asset identity cannot interleave.
type atomicListenerAudioStore interface {
	GetListenerEligibleActiveAudio(ctx context.Context, chapterID string) (AudioAsset, error)
}

func (s *Service) getValidatedListenerActiveAudio(ctx context.Context, chapterID string) (AudioAsset, error) {
	if atomic, ok := s.store.(atomicListenerAudioStore); ok {
		asset, err := atomic.GetListenerEligibleActiveAudio(ctx, chapterID)
		if err != nil {
			if errors.Is(err, ErrListenerAudioNotEligible) {
				return AudioAsset{}, err
			}
			if errors.Is(err, ErrAudioAssetNotFound) || errors.Is(err, ErrNarrationNotFound) {
				return AudioAsset{}, ErrListenerAudioNotEligible
			}
			return AudioAsset{}, err
		}
		return asset, nil
	}

	if err := s.AssertActiveAudioMatchesLatestNarration(ctx, chapterID); err != nil {
		return AudioAsset{}, err
	}
	return s.GetActiveAudioAsset(ctx, chapterID)
}

func (s *Service) presignAudioAsset(ctx context.Context, asset AudioAsset) (string, error) {
	if s.presigner == nil {
		return "", errors.New("presigner not configured")
	}
	return s.presigner.PresignedGetObject(ctx, asset.StorageKey, PresignExpiry)
}
