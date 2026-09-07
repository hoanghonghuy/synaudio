package audio

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// objectDeleter is an optional cleanup capability implemented by production
// object storage. Attempt-unique object keys keep orphaned objects harmless even
// when cleanup itself is unavailable or fails.
type objectDeleter interface {
	Delete(ctx context.Context, key string) error
}

// SynthesizeNarration runs the full audio pipeline for a narration revision:
// segment the script, synthesize each segment, stage each object to bounded local
// files, finalize through the file-oriented processor, stream-upload the result,
// and only then register READY metadata.
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
	fileStorage, ok := s.objectStorage.(FileObjectStorage)
	if !ok {
		return AudioAsset{}, errors.New("object storage file boundary not configured")
	}
	fileProcessor, ok := s.processor.(FileAudioProcessor)
	if !ok {
		return AudioAsset{}, errors.New("audio processor file boundary not configured")
	}

	nar, err := s.store.GetNarrationRevision(ctx, narrationRevisionID)
	if err != nil {
		return AudioAsset{}, err
	}

	segments, err := s.CreateTTSSegments(ctx, narrationRevisionID)
	if err != nil {
		return AudioAsset{}, err
	}

	dir, err := os.MkdirTemp("", "synaudio-narration-*")
	if err != nil {
		return AudioAsset{}, fmt.Errorf("create narration staging dir: %w", err)
	}
	defer os.RemoveAll(dir)

	inputPaths := make([]string, 0, len(segments))
	for i, seg := range segments {
		if err := ctx.Err(); err != nil {
			return AudioAsset{}, err
		}
		synthesized, err := s.SynthesizeSegment(ctx, seg.ID)
		if err != nil {
			return AudioAsset{}, fmt.Errorf("synthesize segment %d: %w", seg.SegmentNo, err)
		}
		path := filepath.Join(dir, fmt.Sprintf("segment-%06d.mp3", i))
		if err := fileStorage.DownloadToFile(ctx, synthesized.TempStorageKey, path); err != nil {
			return AudioAsset{}, fmt.Errorf("stage synthesized segment %d: %w", seg.SegmentNo, err)
		}
		inputPaths = append(inputPaths, path)
	}

	outputPath := filepath.Join(dir, "final.mp3")
	durationMs, err := fileProcessor.ProcessFiles(ctx, inputPaths, outputPath)
	if err != nil {
		return AudioAsset{}, fmt.Errorf("process audio: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return AudioAsset{}, err
	}

	checksum, err := ComputeAudioFileChecksum(ctx, outputPath)
	if err != nil {
		return AudioAsset{}, fmt.Errorf("checksum final audio: %w", err)
	}

	// The object is written before READY metadata exists, so its identity must not
	// depend on a version number that has not yet been committed. A UUID-qualified
	// attempt key guarantees concurrent synthesis attempts never overwrite each
	// other; the version is allocated atomically when metadata is inserted.
	assetID := uuid.NewString()
	storageKey := fmt.Sprintf("chapters/%s/audio/attempts/%s.mp3", nar.ChapterID, assetID)
	sizeBytes, err := fileStorage.UploadFile(ctx, storageKey, outputPath)
	if err != nil {
		return AudioAsset{}, fmt.Errorf("persist final audio: %w", err)
	}

	asset := AudioAsset{
		ID:                        assetID,
		ChapterID:                 nar.ChapterID,
		SourceNarrationRevisionID: narrationRevisionID,
		Status:                    "READY",
		StorageKey:                storageKey,
		MimeType:                  "audio/mpeg",
		SizeBytes:                 sizeBytes,
		DurationMs:                durationMs,
		BitrateKbps:               96,
		Checksum:                  checksum,
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
