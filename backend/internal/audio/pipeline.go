package audio

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// SynthesizeNarration runs the full audio pipeline for a narration revision:
// segment the script, synthesize each segment, concatenate via the audio
// processor, and register a new audio asset version.
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

	processor := s.processor
	if processor == nil {
		processor = NewMockAudioProcessor()
	}

	segAudio := make([]SegmentAudio, 0, len(segments))
	for _, seg := range segments {
		synthesized, err := s.SynthesizeSegment(ctx, seg.ID)
		if err != nil {
			return AudioAsset{}, fmt.Errorf("synthesize segment %d: %w", seg.SegmentNo, err)
		}
		segAudio = append(segAudio, SegmentAudio{
			Data:       []byte(synthesized.TempStorageKey),
			DurationMs: synthesized.DurationMs,
		})
	}

	processed, err := processor.Process(ctx, ProcessInput{Segments: segAudio})
	if err != nil {
		return AudioAsset{}, fmt.Errorf("process audio: %w", err)
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
		SizeBytes:                 int64(len(processed.Data)),
		DurationMs:                processed.DurationMs,
		BitrateKbps:               96,
		IsActive:                  false,
	}

	return s.store.CreateAudioAsset(ctx, asset)
}
