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
)

// Store is the persistence boundary for the story service.
type Store interface {
	CreateStory(ctx context.Context, s Story) (Story, error)
	CreateGenerationPolicy(ctx context.Context, p GenerationPolicy) error
	SlugExists(ctx context.Context, slug string) (bool, error)
}

type Story struct {
	ID          string
	Slug        string
	Title       string
	Description string
	Status      string
	Visibility  string
	CreatedBy   string
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
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
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
