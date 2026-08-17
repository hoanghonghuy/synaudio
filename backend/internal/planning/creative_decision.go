package planning

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrCreativeDecisionNotFound = errors.New("creative decision not found")
)

// CreativeDecision is a controlled creative choice that may require admin input.
type CreativeDecision struct {
	ID                  string
	StoryID             string
	ChapterID           string
	ArcID               string
	Origin              string
	DecisionType        string
	Severity            string
	Status              string
	BlockingLevel       string
	Question            string
	ContextSummary      string
	RecommendedOptionID string
	SelectedOptionID    string
	CustomSelectedText  string
	RejectionScope      string
	RevisitCondition    map[string]any
	TriggeredByRunID    string
	CreatedBy           string
	SelectedBy          string
}

// CreateCreativeDecisionInput is the input for creating a creative decision.
type CreateCreativeDecisionInput struct {
	StoryID        string
	ChapterID      string
	ArcID          string
	Origin         string
	DecisionType   string
	Severity       string
	BlockingLevel  string
	Question       string
	ContextSummary string
	CreatedBy      string
}

// CreateCreativeDecision proposes a new creative decision.
func (s *Service) CreateCreativeDecision(ctx context.Context, in CreateCreativeDecisionInput) (CreativeDecision, error) {
	if in.Question == "" {
		return CreativeDecision{}, errors.New("question must not be empty")
	}
	if in.StoryID == "" {
		return CreativeDecision{}, errors.New("story id must not be empty")
	}

	severity := in.Severity
	if severity == "" {
		severity = "SIGNIFICANT"
	}
	blocking := in.BlockingLevel
	if blocking == "" {
		blocking = "NON_BLOCKING"
	}
	origin := in.Origin
	if origin == "" {
		origin = "AI"
	}

	d := CreativeDecision{
		ID:             uuid.NewString(),
		StoryID:        in.StoryID,
		ChapterID:      in.ChapterID,
		ArcID:          in.ArcID,
		Origin:         origin,
		DecisionType:   in.DecisionType,
		Severity:       severity,
		Status:         "PROPOSED",
		BlockingLevel:  blocking,
		Question:       in.Question,
		ContextSummary: in.ContextSummary,
		CreatedBy:      in.CreatedBy,
	}

	return s.store.CreateCreativeDecision(ctx, d)
}

// SelectCreativeDecision marks a decision as SELECTED by an admin.
func (s *Service) SelectCreativeDecision(ctx context.Context, id, selectedBy string) (CreativeDecision, error) {
	d, err := s.store.GetCreativeDecision(ctx, id)
	if err != nil {
		return CreativeDecision{}, err
	}

	d.Status = "SELECTED"
	d.SelectedBy = selectedBy

	return s.store.UpdateCreativeDecision(ctx, d)
}

// RejectCreativeDecision marks a decision as REJECTED.
func (s *Service) RejectCreativeDecision(ctx context.Context, id, rejectedBy, scope string) (CreativeDecision, error) {
	d, err := s.store.GetCreativeDecision(ctx, id)
	if err != nil {
		return CreativeDecision{}, err
	}

	d.Status = "REJECTED"
	d.RejectionScope = scope
	d.SelectedBy = rejectedBy

	return s.store.UpdateCreativeDecision(ctx, d)
}

// ListCreativeDecisions returns all decisions for a story.
func (s *Service) ListCreativeDecisions(ctx context.Context, storyID string) ([]CreativeDecision, error) {
	return s.store.ListCreativeDecisions(ctx, storyID)
}
