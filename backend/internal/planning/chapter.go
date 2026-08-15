package planning

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrChapterNotFound = errors.New("chapter not found")
)

// Chapter is a first-class production unit.
type Chapter struct {
	ID            string
	StoryID       string
	ChapterNumber int
	Title         string
	Status        string
	ArcID         string
}

// ChapterPlanRevision is a versioned Chapter Plan.
type ChapterPlanRevision struct {
	ID         string
	ChapterID  string
	RevisionNo int
	Plan       map[string]any
	SourceType string
	CreatedBy  string
}

// CreateChapter creates a new Chapter with the next sequential number.
func (s *Service) CreateChapter(ctx context.Context, storyID, title, createdBy string) (Chapter, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Chapter{}, errors.New("chapter title must not be empty")
	}

	number, err := s.store.NextChapterNumber(ctx, storyID)
	if err != nil {
		return Chapter{}, err
	}

	c := Chapter{
		ID:            uuid.NewString(),
		StoryID:       storyID,
		ChapterNumber: number,
		Title:         title,
		Status:        "DRAFT",
	}

	return s.store.CreateChapter(ctx, c)
}

// CreatePlanRevision creates a new versioned Chapter Plan.
func (s *Service) CreatePlanRevision(ctx context.Context, chapterID string, plan map[string]any, createdBy string) (ChapterPlanRevision, error) {
	if len(plan) == 0 {
		return ChapterPlanRevision{}, errors.New("plan content must not be empty")
	}

	revisionNo, err := s.store.NextPlanRevision(ctx, chapterID)
	if err != nil {
		return ChapterPlanRevision{}, err
	}

	p := ChapterPlanRevision{
		ID:         uuid.NewString(),
		ChapterID:  chapterID,
		RevisionNo: revisionNo,
		Plan:       plan,
		SourceType: "AI_GENERATED",
		CreatedBy:  createdBy,
	}

	return s.store.CreatePlanRevision(ctx, p)
}

// ListChapters returns all chapters for a story, ordered by number.
func (s *Service) ListChapters(ctx context.Context, storyID string) ([]Chapter, error) {
	return s.store.ListChapters(ctx, storyID)
}

// GetChapter returns a single chapter by ID.
func (s *Service) GetChapter(ctx context.Context, chapterID string) (Chapter, error) {
	return s.store.GetChapter(ctx, chapterID)
}
