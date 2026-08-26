package planning

import (
	"context"
	"testing"
)

func TestRepairCanonDataSupersedesWrongFact(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, WithMemoryExtractor(&repairExtractor{}))

	// Seed an existing (wrong) fact.
	oldFact, _ := svc.CreateFact(context.Background(), "s1", "CHARACTER", "c1", "HAS_ITEM", map[string]any{"item": "key"}, "NORMAL")

	res, err := svc.RepairCanonData(context.Background(), RepairCanonInput{
		StoryID:           "s1",
		BranchID:          "b1",
		SourceChapterID:   "ch1",
		ContentRevisionID: "cr1",
		CommittedBy:       "u1",
	})
	if err != nil {
		t.Fatalf("repair canon data: %v", err)
	}

	if res.Version.Status != "OFFICIAL" {
		t.Fatalf("expected OFFICIAL version, got %q", res.Version.Status)
	}

	// The old fact should now be superseded.
	updated, err := store.GetFact(context.Background(), oldFact.ID)
	if err != nil {
		t.Fatalf("get fact: %v", err)
	}
	if updated.Status != "SUPERSEDED" {
		t.Fatalf("expected SUPERSEDED, got %q", updated.Status)
	}
}

func TestRepairCanonDataRequiresExtractor(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.RepairCanonData(context.Background(), RepairCanonInput{
		StoryID: "s1",
	}); err == nil {
		t.Fatal("expected error when memory extractor not configured")
	}
}

type repairExtractor struct{}

func (repairExtractor) ExtractMemory(_ context.Context, _ MemoryExtractionInput) (MemoryExtraction, error) {
	return MemoryExtraction{
		Facts: []ExtractedFact{
			{SubjectType: "CHARACTER", SubjectID: "c1", FactType: "HAS_ITEM", Value: map[string]any{"item": "none"}},
		},
	}, nil
}
