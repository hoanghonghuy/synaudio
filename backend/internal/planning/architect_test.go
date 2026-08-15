package planning

import (
	"context"
	"testing"
)

type mockArchitect struct{}

func (mockArchitect) ProposeFoundation(ctx context.Context, in FoundationInput) (FoundationProposal, error) {
	return FoundationProposal{
		Bible: map[string]any{
			"premise": in.Premise,
			"tone":    "adventurous",
		},
		Ending: map[string]any{
			"ending": "resolved",
		},
		Arcs: []map[string]any{
			{"objective": "introduction"},
			{"objective": "climax"},
		},
		Characters: []CharacterProposal{
			{Name: "Hero", Importance: "MAJOR", Profile: map[string]any{"role": "protagonist"}},
		},
	}, nil
}

func TestGenerateFoundationCreatesAllArtifacts(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, WithArchitect(mockArchitect{}))

	res, err := svc.GenerateFoundation(context.Background(), FoundationInput{
		StoryID:   "s1",
		Premise:   "A hero's journey",
		CreatedBy: "u1",
	})
	if err != nil {
		t.Fatalf("generate foundation: %v", err)
	}

	if res.Bible.VersionNo != 1 {
		t.Fatalf("expected bible version 1, got %d", res.Bible.VersionNo)
	}
	if res.Ending.VersionNo != 1 {
		t.Fatalf("expected ending version 1, got %d", res.Ending.VersionNo)
	}
	if len(res.Arcs) != 2 {
		t.Fatalf("expected 2 arcs, got %d", len(res.Arcs))
	}
	if len(res.Characters) != 1 {
		t.Fatalf("expected 1 character, got %d", len(res.Characters))
	}
}

func TestGenerateFoundationRequiresArchitect(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.GenerateFoundation(context.Background(), FoundationInput{
		StoryID: "s1",
		Premise: "A hero's journey",
	}); err == nil {
		t.Fatal("expected error when architect not configured")
	}
}
