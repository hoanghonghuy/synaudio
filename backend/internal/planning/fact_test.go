package planning

import (
	"context"
	"testing"
)

type factFakeStore struct {
	*fakeStore
	facts map[string][]StoryFact
}

func newFactFakeStore() *factFakeStore {
	return &factFakeStore{
		fakeStore: newFakeStore(),
		facts:     map[string][]StoryFact{},
	}
}

func (s *factFakeStore) CreateFact(ctx context.Context, f StoryFact) (StoryFact, error) {
	s.facts[f.StoryID] = append(s.facts[f.StoryID], f)
	return f, nil
}

func (s *factFakeStore) ListFacts(ctx context.Context, storyID string) ([]StoryFact, error) {
	return s.facts[storyID], nil
}

func TestCreateFact(t *testing.T) {
	store := newFactFakeStore()
	svc := NewService(store)

	f, err := svc.CreateFact(context.Background(), "s1", "CHARACTER", "c1", "NAME", map[string]any{"value": "Alice"}, "NORMAL")
	if err != nil {
		t.Fatalf("create fact: %v", err)
	}
	if f.FactType != "NAME" {
		t.Fatalf("expected fact type NAME, got %q", f.FactType)
	}
	if f.Status != "ACTIVE" {
		t.Fatalf("expected ACTIVE, got %q", f.Status)
	}
}

func TestCreateFactRejectsEmptyType(t *testing.T) {
	store := newFactFakeStore()
	svc := NewService(store)

	if _, err := svc.CreateFact(context.Background(), "s1", "CHARACTER", "c1", "  ", map[string]any{}, "NORMAL"); err == nil {
		t.Fatal("expected error for empty fact type")
	}
}

func TestListFactsReturnsAll(t *testing.T) {
	store := newFactFakeStore()
	svc := NewService(store)

	_, _ = svc.CreateFact(context.Background(), "s1", "CHARACTER", "c1", "NAME", map[string]any{"value": "Alice"}, "NORMAL")
	_, _ = svc.CreateFact(context.Background(), "s1", "CHARACTER", "c2", "NAME", map[string]any{"value": "Bob"}, "NORMAL")

	facts, err := svc.ListFacts(context.Background(), "s1")
	if err != nil {
		t.Fatalf("list facts: %v", err)
	}
	if len(facts) != 2 {
		t.Fatalf("expected 2 facts, got %d", len(facts))
	}
}

func TestListFactsReturnsEmpty(t *testing.T) {
	store := newFactFakeStore()
	svc := NewService(store)

	facts, err := svc.ListFacts(context.Background(), "s1")
	if err != nil {
		t.Fatalf("list facts: %v", err)
	}
	if len(facts) != 0 {
		t.Fatalf("expected 0 facts, got %d", len(facts))
	}
}
