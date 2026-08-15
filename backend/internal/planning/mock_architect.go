package planning

import (
	"context"
)

// MockArchitect is a deterministic Story Architect for development/testing.
type MockArchitect struct{}

func NewMockArchitect() *MockArchitect {
	return &MockArchitect{}
}

// ProposeFoundation returns a deterministic foundation proposal.
func (MockArchitect) ProposeFoundation(_ context.Context, in FoundationInput) (FoundationProposal, error) {
	return FoundationProposal{
		Bible: map[string]any{
			"premise": in.Premise,
			"world":   "A fictional world shaped by the premise.",
			"tone":    "adventurous",
			"rules":   []string{"internal consistency"},
		},
		Ending: map[string]any{
			"ending": "resolved",
			"path":   "flexible",
		},
		Arcs: []map[string]any{
			{"objective": "introduction", "conflict": "establishing stakes"},
			{"objective": "rising action", "conflict": "escalating tension"},
			{"objective": "climax", "conflict": "confrontation"},
			{"objective": "resolution", "conflict": "aftermath"},
		},
		Characters: []CharacterProposal{
			{Name: "Protagonist", Importance: "MAJOR", Profile: map[string]any{"role": "protagonist"}},
			{Name: "Antagonist", Importance: "MAJOR", Profile: map[string]any{"role": "antagonist"}},
		},
	}, nil
}
