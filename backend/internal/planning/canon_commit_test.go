package planning

import (
	"context"
	"testing"
)

type commitFakeStore struct {
	*fakeStore
	changeItems map[string][]CanonChangeItem
}

func newCommitFakeStore() *commitFakeStore {
	return &commitFakeStore{
		fakeStore:   newFakeStore(),
		changeItems: map[string][]CanonChangeItem{},
	}
}

func (s *commitFakeStore) CreateCanonChangeItem(ctx context.Context, c CanonChangeItem) (CanonChangeItem, error) {
	s.changeItems[c.CanonVersionID] = append(s.changeItems[c.CanonVersionID], c)
	return c, nil
}

func (s *commitFakeStore) ListCanonChangeItems(ctx context.Context, canonVersionID string) ([]CanonChangeItem, error) {
	return s.changeItems[canonVersionID], nil
}

type mockMemoryExtractor struct {
	facts []ExtractedFact
}

func (m *mockMemoryExtractor) ExtractMemory(_ context.Context, _ MemoryExtractionInput) (MemoryExtraction, error) {
	return MemoryExtraction{Facts: m.facts}, nil
}

func TestCommitCanonCreatesOfficialVersion(t *testing.T) {
	store := newCommitFakeStore()
	svc := NewService(store, WithMemoryExtractor(&mockMemoryExtractor{
		facts: []ExtractedFact{{SubjectType: "CHARACTER", SubjectID: "c1", FactType: "NAME", Value: map[string]any{"value": "Alice"}}},
	}))

	store.branches["s1"] = []CanonBranch{{ID: "b1", StoryID: "s1", Type: "OFFICIAL", Status: "ACTIVE"}}

	res, err := svc.CommitCanon(context.Background(), "s1", "b1", "ch1", "rev1", "u1")
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if res.Version.Status != "OFFICIAL" {
		t.Fatalf("expected OFFICIAL, got %q", res.Version.Status)
	}
	if res.Version.SourceChapterID != "ch1" {
		t.Fatalf("expected source chapter ch1, got %q", res.Version.SourceChapterID)
	}
	if len(res.ChangeItems) != 1 {
		t.Fatalf("expected 1 change item, got %d", len(res.ChangeItems))
	}
}

func TestCommitCanonWithoutExtractorFails(t *testing.T) {
	store := newCommitFakeStore()
	svc := NewService(store)

	store.branches["s1"] = []CanonBranch{{ID: "b1", StoryID: "s1", Type: "OFFICIAL", Status: "ACTIVE"}}

	if _, err := svc.CommitCanon(context.Background(), "s1", "b1", "ch1", "rev1", "u1"); err == nil {
		t.Fatal("expected error when memory extractor not configured")
	}
}

func TestCommitCanonCreatesChangeItemsFromFacts(t *testing.T) {
	store := newCommitFakeStore()
	svc := NewService(store, WithMemoryExtractor(&mockMemoryExtractor{
		facts: []ExtractedFact{
			{SubjectType: "CHARACTER", SubjectID: "c1", FactType: "NAME", Value: map[string]any{"value": "Alice"}},
			{SubjectType: "CHARACTER", SubjectID: "c2", FactType: "NAME", Value: map[string]any{"value": "Bob"}},
		},
	}))

	store.branches["s1"] = []CanonBranch{{ID: "b1", StoryID: "s1", Type: "OFFICIAL", Status: "ACTIVE"}}

	res, err := svc.CommitCanon(context.Background(), "s1", "b1", "ch1", "rev1", "u1")
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if len(res.ChangeItems) != 2 {
		t.Fatalf("expected 2 change items, got %d", len(res.ChangeItems))
	}
}
