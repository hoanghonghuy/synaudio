package planning

import (
	"context"
	"errors"
)

// Architect proposes a Story foundation (Bible, Ending, Arcs, Characters).
type Architect interface {
	ProposeFoundation(ctx context.Context, in FoundationInput) (FoundationProposal, error)
}

// FoundationInput is the input for Story Architect.
type FoundationInput struct {
	StoryID   string
	Premise   string
	CreatedBy string
}

// FoundationProposal is the output of Story Architect.
type FoundationProposal struct {
	Bible      map[string]any
	Ending     map[string]any
	Arcs       []map[string]any
	Characters []CharacterProposal
}

// CharacterProposal is a proposed character.
type CharacterProposal struct {
	Name       string
	Importance string
	Profile    map[string]any
}

// FoundationResult is the result of generating a Story foundation.
type FoundationResult struct {
	Bible      StoryBibleVersion
	Ending     EndingPlanVersion
	Arcs       []StoryArc
	Characters []Character
}

// GenerateFoundation runs the Story Architect and persists the resulting
// foundation artifacts (Bible, Ending, Arcs, Characters).
func (s *Service) GenerateFoundation(ctx context.Context, in FoundationInput) (FoundationResult, error) {
	if s.architect == nil {
		return FoundationResult{}, errors.New("architect not configured")
	}

	proposal, err := s.architect.ProposeFoundation(ctx, in)
	if err != nil {
		return FoundationResult{}, err
	}

	res := FoundationResult{}

	bible, err := s.CreateBibleVersion(ctx, in.StoryID, proposal.Bible, in.CreatedBy)
	if err != nil {
		return FoundationResult{}, err
	}
	res.Bible = bible

	ending, err := s.CreateEndingVersion(ctx, in.StoryID, proposal.Ending, in.CreatedBy)
	if err != nil {
		return FoundationResult{}, err
	}
	res.Ending = ending

	for _, arcContent := range proposal.Arcs {
		arc, err := s.CreateArc(ctx, in.StoryID, arcContent, in.CreatedBy)
		if err != nil {
			return FoundationResult{}, err
		}
		res.Arcs = append(res.Arcs, arc)
	}

	for _, cp := range proposal.Characters {
		c, err := s.CreateCharacter(ctx, in.StoryID, cp.Name, cp.Importance, cp.Profile, in.CreatedBy)
		if err != nil {
			return FoundationResult{}, err
		}
		res.Characters = append(res.Characters, c)
	}

	return res, nil
}
