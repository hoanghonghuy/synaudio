package story

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

const (
	StatusDraft     = "DRAFT"
	StatusActive    = "ACTIVE"
	StatusCompleted = "COMPLETED"
	StatusArchived  = "ARCHIVED"

	VisibilityPrivate = "PRIVATE"
	VisibilityPublic  = "PUBLIC"
)

var (
	ErrInvalidTitle = errors.New("invalid title")
	ErrSlugTaken    = errors.New("slug already taken")
	ErrStoryNotFound = errors.New("story not found")
	ErrContentProfileNotFound = errors.New("content profile not found")
	ErrNotPublicable = errors.New("story not publicable")
)

// Store is the persistence boundary for the story service.
type Store interface {
	CreateStory(ctx context.Context, s Story) (Story, error)
	CreateGenerationPolicy(ctx context.Context, p GenerationPolicy) error
	SlugExists(ctx context.Context, slug string) (bool, error)
	ListGenres(ctx context.Context) ([]Genre, error)
	ListStories(ctx context.Context, publicOnly bool) ([]Story, error)
	GetWorkflowSettings(ctx context.Context, storyID string) (WorkflowSettings, error)
	UpdateWorkflowSettings(ctx context.Context, ws WorkflowSettings) (WorkflowSettings, error)
	NextContentProfileVersion(ctx context.Context, storyID string) (int, error)
	CreateContentProfileVersion(ctx context.Context, cp ContentProfileVersion) (ContentProfileVersion, error)
	GetCurrentContentProfile(ctx context.Context, storyID string) (ContentProfileVersion, error)
	GetStory(ctx context.Context, storyID string) (Story, error)
	UpdateStory(ctx context.Context, s Story) (Story, error)
	CreateStoryAsset(ctx context.Context, a StoryAsset) (StoryAsset, error)
	LinkCoverAsset(ctx context.Context, storyID, assetID string) error
}

// ObjectStorage is the boundary for object storage (MinIO/R2).
type ObjectStorage interface {
	Put(ctx context.Context, key string, data []byte) error
}

const (
	AssetTypeCover = "COVER"

	AssetStatusPending = "PENDING"
	AssetStatusReady   = "READY"
)

type StoryAsset struct {
	ID          string
	StoryID     string
	Type        string
	StorageKey  string
	MimeType    string
	SizeBytes   int64
	Checksum    string
	RightsStatus string
	Status      string
	CreatedBy   string
}

type UploadCoverInput struct {
	StoryID     string
	Filename    string
	ContentType string
	Data        []byte
	CreatedBy   string
}

type Genre struct {
	ID   string
	Slug string
	Name string
}

type ListStoriesInput struct {
	PublicOnly bool
}

type WorkflowSettings struct {
	StoryID             string
	BatchGenerationSize int
	CreativeAutonomy    string
	PreferredTextProvider string
	PreferredTextModel    string
	PreferredTTSProvider  string
	PreferredVoiceID      string
	PauseBeforeTTS        bool
	AutoAIReview          bool
	PlanningHorizon       int
	FallbackPolicy        map[string]any
	UpdatedBy             string
}

type WorkflowSettingsInput struct {
	BatchGenerationSize int
	CreativeAutonomy    string
	PreferredTextProvider string
	PreferredTextModel    string
	PreferredTTSProvider  string
	PreferredVoiceID      string
	PauseBeforeTTS        bool
	AutoAIReview          bool
	PlanningHorizon       int
	FallbackPolicy        map[string]any
}

type ContentProfileVersion struct {
	ID        string
	StoryID   string
	VersionNo int
	Profile   map[string]any
	CreatedBy string
}

type ContentProfileInput struct {
	MaturityTarget string
	AllowedThemes  []string
	DisallowedThemes []string
	ViolenceLevel  string
	LanguageLimits string
	RomanceLimits  string
	Constraints    map[string]any
	CreatedBy      string
}

type Story struct {
	ID                 string
	Slug               string
	Title              string
	Description        string
	Status             string
	Visibility         string
	StatusBeforeArchive string
	CoverAssetID       string
	CreatedBy          string
}

type GenerationPolicy struct {
	StoryID                 string
	MinimumAudioDurationSec int
	TargetAudioDurationSec  int
	ContentOrigin           string
	Language                string
	NarrationLanguage       string
	PolicyVersion           int
	CreatedBy               string
}

type GenerationPolicyInput struct {
	MinimumAudioDurationSec int
	TargetAudioDurationSec  int
	ContentOrigin           string
	Language                string
	NarrationLanguage       string
}

type CreateStoryInput struct {
	Title       string
	Description string
	CreatedBy   string
	Policy      GenerationPolicyInput
}

type Service struct {
	store   Store
	storage ObjectStorage
}

type Option func(*Service)

func WithObjectStorage(s ObjectStorage) Option {
	return func(svc *Service) {
		svc.storage = s
	}
}

func NewService(store Store, opts ...Option) *Service {
	svc := &Service{store: store}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// CreateStory creates a DRAFT/PRIVATE story with an immutable generation policy.
func (s *Service) CreateStory(ctx context.Context, in CreateStoryInput) (Story, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return Story{}, ErrInvalidTitle
	}

	slug := Slugify(title)
	exists, err := s.store.SlugExists(ctx, slug)
	if err != nil {
		return Story{}, err
	}
	if exists {
		return Story{}, ErrSlugTaken
	}

	st := Story{
		ID:          uuid.NewString(),
		Slug:        slug,
		Title:       title,
		Description: strings.TrimSpace(in.Description),
		Status:      StatusDraft,
		Visibility:  VisibilityPrivate,
		CreatedBy:   in.CreatedBy,
	}

	created, err := s.store.CreateStory(ctx, st)
	if err != nil {
		return Story{}, err
	}

	policy := GenerationPolicy{
		StoryID:                 created.ID,
		MinimumAudioDurationSec: in.Policy.MinimumAudioDurationSec,
		TargetAudioDurationSec:  in.Policy.TargetAudioDurationSec,
		ContentOrigin:           in.Policy.ContentOrigin,
		Language:                in.Policy.Language,
		NarrationLanguage:       in.Policy.NarrationLanguage,
		PolicyVersion:           1,
		CreatedBy:               in.CreatedBy,
	}
	if err := s.store.CreateGenerationPolicy(ctx, policy); err != nil {
		return Story{}, err
	}

	return created, nil
}

// ListGenres returns all available genres.
func (s *Service) ListGenres(ctx context.Context) ([]Genre, error) {
	return s.store.ListGenres(ctx)
}

// ListStories returns stories, optionally filtered to public only.
func (s *Service) ListStories(ctx context.Context, in ListStoriesInput) ([]Story, error) {
	return s.store.ListStories(ctx, in.PublicOnly)
}

// GetWorkflowSettings returns the mutable workflow settings for a story.
func (s *Service) GetWorkflowSettings(ctx context.Context, storyID string) (WorkflowSettings, error) {
	return s.store.GetWorkflowSettings(ctx, storyID)
}

// UpdateWorkflowSettings updates the mutable workflow settings for a story.
func (s *Service) UpdateWorkflowSettings(ctx context.Context, storyID string, in WorkflowSettingsInput) (WorkflowSettings, error) {
	ws := WorkflowSettings{
		StoryID:              storyID,
		BatchGenerationSize:  in.BatchGenerationSize,
		CreativeAutonomy:     in.CreativeAutonomy,
		PreferredTextProvider: in.PreferredTextProvider,
		PreferredTextModel:    in.PreferredTextModel,
		PreferredTTSProvider:  in.PreferredTTSProvider,
		PreferredVoiceID:      in.PreferredVoiceID,
		PauseBeforeTTS:        in.PauseBeforeTTS,
		AutoAIReview:          in.AutoAIReview,
		PlanningHorizon:       in.PlanningHorizon,
		FallbackPolicy:        in.FallbackPolicy,
	}
	return s.store.UpdateWorkflowSettings(ctx, ws)
}

// CreateContentProfileVersion creates a new versioned content profile.
func (s *Service) CreateContentProfileVersion(ctx context.Context, storyID string, in ContentProfileInput) (ContentProfileVersion, error) {
	versionNo, err := s.store.NextContentProfileVersion(ctx, storyID)
	if err != nil {
		return ContentProfileVersion{}, err
	}

	profile := map[string]any{
		"maturity_target":    in.MaturityTarget,
		"allowed_themes":     in.AllowedThemes,
		"disallowed_themes":  in.DisallowedThemes,
		"violence_level":     in.ViolenceLevel,
		"language_limits":    in.LanguageLimits,
		"romance_limits":     in.RomanceLimits,
		"constraints":        in.Constraints,
	}

	cp := ContentProfileVersion{
		ID:        uuid.NewString(),
		StoryID:   storyID,
		VersionNo: versionNo,
		Profile:   profile,
		CreatedBy: in.CreatedBy,
	}

	return s.store.CreateContentProfileVersion(ctx, cp)
}

// GetCurrentContentProfile returns the latest content profile version.
func (s *Service) GetCurrentContentProfile(ctx context.Context, storyID string) (ContentProfileVersion, error) {
	return s.store.GetCurrentContentProfile(ctx, storyID)
}

// ActivateStory transitions a DRAFT story to ACTIVE.
func (s *Service) ActivateStory(ctx context.Context, storyID string) (Story, error) {
	st, err := s.store.GetStory(ctx, storyID)
	if err != nil {
		return Story{}, err
	}
	st.Status = StatusActive
	return s.store.UpdateStory(ctx, st)
}

// ArchiveStory transitions a story to ARCHIVED, saving the previous status.
func (s *Service) ArchiveStory(ctx context.Context, storyID string) (Story, error) {
	st, err := s.store.GetStory(ctx, storyID)
	if err != nil {
		return Story{}, err
	}
	st.StatusBeforeArchive = st.Status
	st.Status = StatusArchived
	return s.store.UpdateStory(ctx, st)
}

// RestoreStory returns an archived story to its previous status.
func (s *Service) RestoreStory(ctx context.Context, storyID string) (Story, error) {
	st, err := s.store.GetStory(ctx, storyID)
	if err != nil {
		return Story{}, err
	}
	if st.StatusBeforeArchive != "" {
		st.Status = st.StatusBeforeArchive
		st.StatusBeforeArchive = ""
	}
	return s.store.UpdateStory(ctx, st)
}

// MakePublic sets a story to PUBLIC, requiring ACTIVE or COMPLETED status.
func (s *Service) MakePublic(ctx context.Context, storyID string) (Story, error) {
	st, err := s.store.GetStory(ctx, storyID)
	if err != nil {
		return Story{}, err
	}
	if st.Status != StatusActive && st.Status != StatusCompleted {
		return Story{}, ErrNotPublicable
	}
	st.Visibility = VisibilityPublic
	return s.store.UpdateStory(ctx, st)
}

// MakePrivate sets a story to PRIVATE.
func (s *Service) MakePrivate(ctx context.Context, storyID string) (Story, error) {
	st, err := s.store.GetStory(ctx, storyID)
	if err != nil {
		return Story{}, err
	}
	st.Visibility = VisibilityPrivate
	return s.store.UpdateStory(ctx, st)
}

// UploadCover stores a cover image in object storage and links it to the story.
func (s *Service) UploadCover(ctx context.Context, in UploadCoverInput) (StoryAsset, error) {
	if _, err := s.store.GetStory(ctx, in.StoryID); err != nil {
		return StoryAsset{}, err
	}
	if s.storage == nil {
		return StoryAsset{}, errors.New("object storage not configured")
	}

	assetID := uuid.NewString()
	storageKey := "stories/" + in.StoryID + "/cover/" + assetID

	if err := s.storage.Put(ctx, storageKey, in.Data); err != nil {
		return StoryAsset{}, err
	}

	asset := StoryAsset{
		ID:          assetID,
		StoryID:     in.StoryID,
		Type:        AssetTypeCover,
		StorageKey:  storageKey,
		MimeType:    in.ContentType,
		SizeBytes:   int64(len(in.Data)),
		Status:      AssetStatusReady,
		CreatedBy:   in.CreatedBy,
	}

	created, err := s.store.CreateStoryAsset(ctx, asset)
	if err != nil {
		return StoryAsset{}, err
	}

	if err := s.store.LinkCoverAsset(ctx, in.StoryID, created.ID); err != nil {
		return StoryAsset{}, err
	}

	return created, nil
}

var (
	nonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)
	dashRe     = regexp.MustCompile(`-+`)
)

// Slugify converts a title into a URL-safe slug, transliterating Vietnamese
// diacritics and lowercasing.
func Slugify(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	title = removeDiacritics(title)

	title = nonAlnumRe.ReplaceAllString(title, "-")
	title = dashRe.ReplaceAllString(title, "-")
	title = strings.Trim(title, "-")

	return title
}

var diacritics = map[rune]rune{
	'à': 'a', 'á': 'a', 'ả': 'a', 'ã': 'a', 'ạ': 'a',
	'ă': 'a', 'ằ': 'a', 'ắ': 'a', 'ẳ': 'a', 'ẵ': 'a', 'ặ': 'a',
	'â': 'a', 'ầ': 'a', 'ấ': 'a', 'ẩ': 'a', 'ẫ': 'a', 'ậ': 'a',
	'è': 'e', 'é': 'e', 'ẻ': 'e', 'ẽ': 'e', 'ẹ': 'e',
	'ê': 'e', 'ề': 'e', 'ế': 'e', 'ể': 'e', 'ễ': 'e', 'ệ': 'e',
	'ì': 'i', 'í': 'i', 'ỉ': 'i', 'ĩ': 'i', 'ị': 'i',
	'ò': 'o', 'ó': 'o', 'ỏ': 'o', 'õ': 'o', 'ọ': 'o',
	'ô': 'o', 'ồ': 'o', 'ố': 'o', 'ổ': 'o', 'ỗ': 'o', 'ộ': 'o',
	'ơ': 'o', 'ờ': 'o', 'ớ': 'o', 'ở': 'o', 'ỡ': 'o', 'ợ': 'o',
	'ù': 'u', 'ú': 'u', 'ủ': 'u', 'ũ': 'u', 'ụ': 'u',
	'ư': 'u', 'ừ': 'u', 'ứ': 'u', 'ử': 'u', 'ữ': 'u', 'ự': 'u',
	'ỳ': 'y', 'ý': 'y', 'ỷ': 'y', 'ỹ': 'y', 'ỵ': 'y',
	'đ': 'd',
}

func removeDiacritics(s string) string {
	var b strings.Builder
	for _, r := range s {
		if m, ok := diacritics[r]; ok {
			b.WriteRune(m)
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
