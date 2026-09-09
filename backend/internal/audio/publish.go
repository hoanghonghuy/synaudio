package audio

import (
	"context"
	"errors"
)

// CheckPublishAudioReady reports missing audio dependencies blocking publish.
func (s *Service) CheckPublishAudioReady(ctx context.Context, chapterID string) ([]string, error) {
	missing := []string{}

	nar, err := s.GetLatestNarrationRevision(ctx, chapterID)
	if err != nil {
		if errors.Is(err, ErrNarrationNotFound) {
			return []string{"narration"}, nil
		}
		return nil, err
	}

	if s.approvedContent == nil {
		missing = append(missing, "approved_content_authority")
	} else if err := s.approvedContent.RequireApprovedContentRevision(ctx, chapterID, nar.SourceContentRevisionID); err != nil {
		missing = append(missing, "approved_content")
	}

	asset, err := s.GetActiveAudioAsset(ctx, chapterID)
	if err != nil {
		if errors.Is(err, ErrAudioAssetNotFound) {
			missing = append(missing, "active_audio")
			return missing, nil
		}
		return nil, err
	}
	if asset.Status != "READY" {
		missing = append(missing, "active_audio")
	}
	if asset.SourceNarrationRevisionID != nar.ID {
		missing = append(missing, "active_audio_stale")
	}

	return missing, nil
}
