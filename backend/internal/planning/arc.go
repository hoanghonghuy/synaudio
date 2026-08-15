package planning

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrArcNotFound = errors.New("arc not found")
)

// StoryArc is a first-class, stable-identity planning artifact.
type StoryArc struct {
	ID               string
	StoryID          string
	Ordinal          int
	Status           string
	CurrentVersionID string
}

// ArcVersion is a versioned Arc content.
type ArcVersion struct {
	ID        string
	ArcID     string
	VersionNo int
	Content   map[string]any
	CreatedBy string
}

// CreateArc creates a new Arc with the next ordinal and an initial version.
func (s *Service) CreateArc(ctx context.Context, storyID string, content map[string]any, createdBy string) (StoryArc, error) {
	if len(content) == 0 {
		return StoryArc{}, errors.New("arc content must not be empty")
	}

	ordinal, err := s.store.NextArcOrdinal(ctx, storyID)
	if err != nil {
		return StoryArc{}, err
	}

	arc := StoryArc{
		ID:      uuid.NewString(),
		StoryID: storyID,
		Ordinal: ordinal,
		Status:  "PLANNED",
	}

	created, err := s.store.CreateArc(ctx, arc)
	if err != nil {
		return StoryArc{}, err
	}

	versionNo, err := s.store.NextArcVersion(ctx, created.ID)
	if err != nil {
		return StoryArc{}, err
	}

	v := ArcVersion{
		ID:        uuid.NewString(),
		ArcID:     created.ID,
		VersionNo: versionNo,
		Content:   content,
		CreatedBy: createdBy,
	}

	if _, err := s.store.CreateArcVersion(ctx, v); err != nil {
		return StoryArc{}, err
	}

	return created, nil
}

// ListArcs returns all arcs for a story, ordered by ordinal.
func (s *Service) ListArcs(ctx context.Context, storyID string) ([]StoryArc, error) {
	return s.store.ListArcs(ctx, storyID)
}

// GetArc returns a single arc by ID.
func (s *Service) GetArc(ctx context.Context, arcID string) (StoryArc, error) {
	return s.store.GetArc(ctx, arcID)
}
