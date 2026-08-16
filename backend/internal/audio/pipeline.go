package audio

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// SynthesizeNarration runs the full audio pipeline for a narration revision:
// segment the script, synthesize each segment, concatenate, and register
// a new audio asset version.
func (s *Service) SynthesizeNarration(ctx context.Context, narrationRevisionID string) (AudioAsset, error) {
	if s.tts == nil {
		return AudioAsset{}, errors.New("tts not configured")
	}

	nar, err := s.store.GetNarrationRevision(ctx, narrationRevisionID)
	if err != nil {
		return AudioAsset{}, err
	}

	segments, err := s.CreateTTSSegments(ctx, narrationRevisionID)
	if err != nil {
		return AudioAsset{}, err
	}

	totalDurationMs := 0
	for _, seg := range segments {
		synthesized, err := s.SynthesizeSegment(ctx, seg.ID)
		if err != nil {
			return AudioAsset{}, fmt.Errorf("synthesize segment %d: %w", seg.SegmentNo, err)
		}
		totalDurationMs += synthesized.DurationMs
	}

	versionNo, err := s.store.NextAudioVersion(ctx, nar.ChapterID)
	if err != nil {
		return AudioAsset{}, err
	}

	asset := AudioAsset{
		ID:                        uuid.NewString(),
		ChapterID:                 nar.ChapterID,
		VersionNo:                 versionNo,
		SourceNarrationRevisionID: narrationRevisionID,
		Status:                    "READY",
		StorageKey:                fmt.Sprintf("chapters/%s/audio/v%d/chapter.mp3", nar.ChapterID, versionNo),
		MimeType:                  "audio/mpeg",
		SizeBytes:                 int64(totalDurationMs),
		DurationMs:                totalDurationMs,
		BitrateKbps:               96,
		IsActive:                  false,
	}

	return s.store.CreateAudioAsset(ctx, asset)
}
