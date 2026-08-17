package planning

import (
	"context"
	"testing"
)

func TestCreateProvisionalCanonVersion(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	store.branches["s1"] = []CanonBranch{{ID: "pb1", StoryID: "s1", Type: "PROVISIONAL", Status: "ACTIVE"}}

	v, err := svc.CreateProvisionalCanonVersion(context.Background(), "s1", "pb1", "ch1", "rev1")
	if err != nil {
		t.Fatalf("create provisional version: %v", err)
	}
	if v.Status != "PROVISIONAL" {
		t.Fatalf("expected PROVISIONAL, got %q", v.Status)
	}
	if v.SourceChapterID != "ch1" {
		t.Fatalf("expected source chapter ch1, got %q", v.SourceChapterID)
	}
	if v.SourceContentRevisionID != "rev1" {
		t.Fatalf("expected source revision rev1, got %q", v.SourceContentRevisionID)
	}
}

func TestPromoteProvisionalToOfficial(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	store.branches["s1"] = []CanonBranch{{ID: "pb1", StoryID: "s1", Type: "PROVISIONAL", Status: "ACTIVE"}}

	v, _ := svc.CreateProvisionalCanonVersion(context.Background(), "s1", "pb1", "ch1", "rev1")

	promoted, err := svc.PromoteProvisionalVersion(context.Background(), v.ID, "u1")
	if err != nil {
		t.Fatalf("promote version: %v", err)
	}
	if promoted.Status != "OFFICIAL" {
		t.Fatalf("expected OFFICIAL, got %q", promoted.Status)
	}
	if promoted.CommittedBy != "u1" {
		t.Fatalf("expected committed_by u1, got %q", promoted.CommittedBy)
	}
}

func TestPromoteProvisionalNotFound(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.PromoteProvisionalVersion(context.Background(), "missing", "u1"); err == nil {
		t.Fatal("expected error for missing version")
	}
}
