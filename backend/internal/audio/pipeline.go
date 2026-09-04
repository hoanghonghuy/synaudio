package audio

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// objectDeleter is an optional cleanup capability implemented by production
// object storage. Attempt-unique object keys keep orphaned objects harmless even
// when cleanup itself is unavailable or fails.
type objectDeleter interface {
	Delete(ctx context.Context, key string) error
}

// SynthesizeNarration runs the full audio pipeline for a narration revision:
// segment the script, synthesize each segment, concatenate via the audio
// processor, and register a new audio asset version.
func (s *Service) SynthesizeNarration(ctx context.Context, narrationRevisionID string) (AudioAsset, error) {
	if s.tts == nil {
		return AudioAsset{}, errors.New("tts not configured")
	}
	if s.objectStorage == nil {
		return AudioAsset{}, errors.New("object storage not configured")
	}
	if s.processor == nil {
		return AudioAsset{}, errors.New("audio processor not configured")
	}

	nar, err := s.store.GetNarrationRevision(ctx, narrationRevisionID)
	if err != nil {
		return AudioAsset{}, err
	}

	segments, err := s.CreateTTSSegments(ctx, narrationRevisionID)
	if err != nil {
		return AudioAsset{}, err
	}

	segAudio := make([]SegmentAudio, 0, len(segments))
	for _, seg := range segments {
		synthesized, err := s.SynthesizeSegment(ctx, seg.ID)
		if err != nil {
			return AudioAsset{}, fmt.Errorf("synthesize segment %d: %w", seg.SegmentNo, err)
		}
		data, err := s.objectStorage.Get(ctx, synthesized.TempStorageKey)
		if err != nil {
			return AudioAsset{}, fmt.Errorf("load synthesized segment %d: %w", seg.SegmentNo, err)
		}
		segAudio = append(segAudio, SegmentAudio{
			Data:       data,
			DurationMs: synthesized.DurationMs,
		})
	}

	processed, err := s.processor.Process(ctx, ProcessInput{Segments: segAudio})
	if err != nil {
		return AudioAsset{}, fmt.Errorf("process audio: %w", err)
	}

	// The object is written before READY metadata exists, so its identity must not
	// depend on a version number that has not yet been committed. A UUID-qualified
	// attempt key guarantees concurrent synthesis attempts never overwrite each
	// other; the version is allocated atomically when metadata is inserted.
	assetID := uuid.NewString()
	storageKey := fmt.Sprintf("chapters/%s/audio/attempts/%s.mp3", nar.ChapterID, assetID)
	if err := s.objectStorage.Put(ctx, storageKey, processed.Data); err != nil {
		return AudioAsset{}, fmt.Errorf("persist final audio: %w", err)
	}

	asset := AudioAsset{
		ID:                        assetID,
		ChapterID:                 nar.ChapterID,
		SourceNarrationRevisionID: narrationRevisionID,
		Status:                    "READY",
		StorageKey:                storageKey,
		MimeType:                  "audio/mpeg",
		SizeBytes:                 int64(len(processed.Data)),
		DurationMs:                processed.DurationMs,
		BitrateKbps:               96,
		IsActive:                  false,
	}

	persisted, err := s.persistAudioAsset(ctx, asset)
	if err == nil {
		return persisted, nil
	}

	// Metadata is the READY authority. If metadata registration fails after the
	// object write, remove only this attempt's unique object. If cleanup fails,
	// the UUID-qualified key still guarantees the orphan cannot overwrite or be
	// mistaken for a later valid asset.
	if cleaner, ok := s.objectStorage.(objectDeleter); ok {
		if cleanupErr := cleaner.Delete(ctx, storageKey); cleanupErr != nil {
			return AudioAsset{}, fmt.Errorf("register audio asset: %w (cleanup %q failed: %v)", err, storageKey, cleanupErr)
		}
	}
	return AudioAsset{}, fmt.Errorf("register audio asset: %w", err)
}
