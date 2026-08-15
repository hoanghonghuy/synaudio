package audio

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrNarrationNotFound = errors.New("narration revision not found")
	ErrAudioAssetNotFound = errors.New("audio asset not found")
)

// NarrationRevision is a versioned narration script for a chapter.
type NarrationRevision struct {
	ID                      string
	ChapterID               string
	RevisionNo              int
	SourceContentRevisionID string
	VoiceID                 string
	Script                  string
	Status                  string
	CreatedBy               string
}

// AudioAsset is a versioned, encoded audio file for a chapter.
type AudioAsset struct {
	ID                        string
	ChapterID                 string
	VersionNo                 int
	SourceNarrationRevisionID string
	Status                    string
	StorageKey                string
	MimeType                  string
	SizeBytes                 int64
	DurationMs                int
	BitrateKbps               int
	Checksum                  string
	IsActive                  bool
}

// Store is the persistence boundary for the audio service.
type Store interface {
	NextNarrationRevision(ctx context.Context, chapterID string) (int, error)
	CreateNarrationRevision(ctx context.Context, r NarrationRevision) (NarrationRevision, error)
	GetNarrationRevision(ctx context.Context, revisionID string) (NarrationRevision, error)

	NextAudioVersion(ctx context.Context, chapterID string) (int, error)
	CreateAudioAsset(ctx context.Context, a AudioAsset) (AudioAsset, error)
	GetAudioAsset(ctx context.Context, assetID string) (AudioAsset, error)
	GetActiveAudioAsset(ctx context.Context, chapterID string) (AudioAsset, error)
	SetActiveAudioAsset(ctx context.Context, chapterID, assetID string) (AudioAsset, error)
}

// Service orchestrates narration and audio asset production.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateNarrationRevision creates a new narration revision for a chapter.
func (s *Service) CreateNarrationRevision(ctx context.Context, chapterID, sourceContentRevisionID, voiceID, script, createdBy string) (NarrationRevision, error) {
	script = strings.TrimSpace(script)
	if script == "" {
		return NarrationRevision{}, errors.New("script must not be empty")
	}

	revisionNo, err := s.store.NextNarrationRevision(ctx, chapterID)
	if err != nil {
		return NarrationRevision{}, err
	}

	r := NarrationRevision{
		ID:                      uuid.NewString(),
		ChapterID:               chapterID,
		RevisionNo:              revisionNo,
		SourceContentRevisionID: sourceContentRevisionID,
		VoiceID:                 voiceID,
		Script:                  script,
		Status:                  "DRAFT",
		CreatedBy:               createdBy,
	}

	return s.store.CreateNarrationRevision(ctx, r)
}

// CreateAudioAsset creates a new audio asset version for a chapter.
func (s *Service) CreateAudioAsset(ctx context.Context, chapterID, sourceNarrationRevisionID, storageKey, mimeType string, sizeBytes int64, durationMs, bitrateKbps int) (AudioAsset, error) {
	versionNo, err := s.store.NextAudioVersion(ctx, chapterID)
	if err != nil {
		return AudioAsset{}, err
	}

	a := AudioAsset{
		ID:                        uuid.NewString(),
		ChapterID:                 chapterID,
		VersionNo:                 versionNo,
		SourceNarrationRevisionID: sourceNarrationRevisionID,
		Status:                    "READY",
		StorageKey:                storageKey,
		MimeType:                  mimeType,
		SizeBytes:                 sizeBytes,
		DurationMs:                durationMs,
		BitrateKbps:               bitrateKbps,
		IsActive:                  false,
	}

	return s.store.CreateAudioAsset(ctx, a)
}

// ActivateAudioAsset atomically promotes an asset to active for its chapter.
func (s *Service) ActivateAudioAsset(ctx context.Context, chapterID, assetID string) (AudioAsset, error) {
	if _, err := s.store.GetAudioAsset(ctx, assetID); err != nil {
		return AudioAsset{}, err
	}
	return s.store.SetActiveAudioAsset(ctx, chapterID, assetID)
}

// GetActiveAudioAsset returns the currently active audio asset for a chapter.
func (s *Service) GetActiveAudioAsset(ctx context.Context, chapterID string) (AudioAsset, error) {
	return s.store.GetActiveAudioAsset(ctx, chapterID)
}
