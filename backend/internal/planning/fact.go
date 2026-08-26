package planning

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// StoryFact is a canonical fact with provenance and status.
type StoryFact struct {
	ID               string
	StoryID          string
	SubjectType      string
	SubjectID        string
	FactType         string
	Value            map[string]any
	Importance       string
	Status           string
	SupersedesFactID string
}

// CreateFact creates a new ACTIVE StoryFact.
func (s *Service) CreateFact(ctx context.Context, storyID, subjectType, subjectID, factType string, value map[string]any, importance string) (StoryFact, error) {
	factType = strings.TrimSpace(factType)
	if factType == "" {
		return StoryFact{}, errors.New("fact type must not be empty")
	}
	if importance == "" {
		importance = "NORMAL"
	}

	f := StoryFact{
		ID:          uuid.NewString(),
		StoryID:     storyID,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		FactType:    factType,
		Value:       value,
		Importance:  importance,
		Status:      "ACTIVE",
	}

	return s.store.CreateFact(ctx, f)
}

// ListFacts returns all facts for a story.
func (s *Service) ListFacts(ctx context.Context, storyID string) ([]StoryFact, error) {
	return s.store.ListFacts(ctx, storyID)
}
