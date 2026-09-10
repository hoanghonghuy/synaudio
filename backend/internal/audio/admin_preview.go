package audio

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrAdminAudioPreviewNotEligible = errors.New("audio asset not eligible for admin preview")
	ErrAudioPresignerRequired       = errors.New("audio presigner not configured")
)

// GetAdminAudioPreviewURL issues a short-lived URL for one explicitly selected
// durable audio asset. Authorization is owned by the /admin HTTP boundary; this
// service method additionally enforces chapter membership and durable READY state
// so an asset ID cannot be substituted across chapters or used to presign an
// incomplete/private object accidentally.
func (s *Service) GetAdminAudioPreviewURL(ctx context.Context, chapterID, assetID string) (string, error) {
	if s.presigner == nil {
		return "", ErrAudioPresignerRequired
	}

	asset, err := s.store.GetAudioAsset(ctx, assetID)
	if err != nil {
		return "", err
	}
	if asset.ChapterID != chapterID {
		// Deliberately conceal cross-chapter membership as not-found semantics.
		return "", ErrAudioAssetNotFound
	}
	if asset.Status != "READY" || strings.TrimSpace(asset.StorageKey) == "" {
		return "", ErrAdminAudioPreviewNotEligible
	}

	return s.presigner.PresignedGetObject(ctx, asset.StorageKey, PresignExpiry)
}
