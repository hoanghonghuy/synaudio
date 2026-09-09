package audio

import (
	"context"
	"errors"
)

// AssertActiveAudioMatchesLatestNarration returns nil only when the chapter has
// an active READY asset sourced from its latest narration revision.
func (s *Service) AssertActiveAudioMatchesLatestNarration(ctx context.Context, chapterID string) error {
	nar, err := s.GetLatestNarrationRevision(ctx, chapterID)
	if err != nil {
		if errors.Is(err, ErrNarrationNotFound) {
			return ErrListenerAudioNotEligible
		}
		return err
	}

	asset, err := s.GetActiveAudioAsset(ctx, chapterID)
	if err != nil {
		if errors.Is(err, ErrAudioAssetNotFound) {
			return ErrListenerAudioNotEligible
		}
		return err
	}
	if asset.Status != "READY" {
		return ErrListenerAudioNotEligible
	}
	if asset.SourceNarrationRevisionID != nar.ID {
		return ErrListenerAudioNotEligible
	}
	return nil
}
