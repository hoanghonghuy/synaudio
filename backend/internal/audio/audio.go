package audio

import (
	"context"
	"errors"
	"strings"
	"time"

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

	CreateTTSSegment(ctx context.Context, seg TTSSegment) (TTSSegment, error)
	GetTTSSegment(ctx context.Context, segmentID string) (TTSSegment, error)
	UpdateTTSSegment(ctx context.Context, seg TTSSegment) (TTSSegment, error)

	NextAudioVersion(ctx context.Context, chapterID string) (int, error)
	CreateAudioAsset(ctx context.Context, a AudioAsset) (AudioAsset, error)
	GetAudioAsset(ctx context.Context, assetID string) (AudioAsset, error)
	GetActiveAudioAsset(ctx context.Context, chapterID string) (AudioAsset, error)
	SetActiveAudioAsset(ctx context.Context, chapterID, assetID string) (AudioAsset, error)
}

// ObjectStorage persists and loads private audio objects by key.
type ObjectStorage interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
}

// Presigner generates presigned download URLs for object storage.
type Presigner interface {
	PresignedGetObject(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// Service orchestrates narration and audio asset production.
type Service struct {
	store         Store
	tts           TTSProvider
	objectStorage ObjectStorage
	presigner     Presigner
	processor     AudioProcessor
}

type Option func(*Service)

func WithTTS(p TTSProvider) Option {
	return func(svc *Service) {
		svc.tts = p
	}
}

func WithObjectStorage(storage ObjectStorage) Option {
	return func(svc *Service) {
		svc.objectStorage = storage
	}
}

func WithPresigner(p Presigner) Option {
	return func(svc *Service) {
		svc.presigner = p
	}
}

func WithAudioProcessor(p AudioProcessor) Option {
	return func(svc *Service) {
		svc.processor = p
	}
}

func NewService(store Store, opts ...Option) *Service {
	svc := &Service{store: store}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
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

// PresignExpiry is how long a presigned audio URL stays valid (spec default: 2 hours).
const PresignExpiry = 2 * time.Hour

// GetAudioURL returns a presigned download URL for the active audio asset.
func (s *Service) GetAudioURL(ctx context.Context, chapterID string) (string, error) {
	if s.presigner == nil {
		return "", errors.New("presigner not configured")
	}

	asset, err := s.store.GetActiveAudioAsset(ctx, chapterID)
	if err != nil {
		return "", err
	}

	return s.presigner.PresignedGetObject(ctx, asset.StorageKey, PresignExpiry)
}
