package listener

import (
	"context"
	"testing"
)

func TestApplyRevisionImpactMarksRelistenRequired(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	// Seed progress for a user who completed the chapter.
	_, _ = svc.SaveProgress(context.Background(), "u1", "ch1", 1000, "a1", "s1")
	_, _ = svc.MarkCompleted(context.Background(), "u1", "ch1")

	affected, err := svc.ApplyRevisionImpact(context.Background(), "ch1", "RELISTEN_REQUIRED")
	if err != nil {
		t.Fatalf("apply revision impact: %v", err)
	}
	if affected != 1 {
		t.Fatalf("expected 1 affected listener, got %d", affected)
	}

	p, err := svc.GetProgress(context.Background(), "u1", "ch1")
	if err != nil {
		t.Fatalf("get progress: %v", err)
	}
	if p.RelistenStatus != "RELISTEN_REQUIRED" {
		t.Fatalf("expected RELISTEN_REQUIRED, got %q", p.RelistenStatus)
	}
}

func TestApplyRevisionImpactNoRelistenNeeded(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	_, _ = svc.SaveProgress(context.Background(), "u1", "ch1", 1000, "a1", "s1")

	affected, err := svc.ApplyRevisionImpact(context.Background(), "ch1", "NO_RELISTEN_NEEDED")
	if err != nil {
		t.Fatalf("apply revision impact: %v", err)
	}
	if affected != 1 {
		t.Fatalf("expected 1 affected listener, got %d", affected)
	}

	p, _ := svc.GetProgress(context.Background(), "u1", "ch1")
	if p.RelistenStatus != "NO_RELISTEN_NEEDED" {
		t.Fatalf("expected NO_RELISTEN_NEEDED, got %q", p.RelistenStatus)
	}
}

func TestApplyRevisionImpactPreservesCompletion(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	_, _ = svc.SaveProgress(context.Background(), "u1", "ch1", 1000, "a1", "s1")
	_, _ = svc.MarkCompleted(context.Background(), "u1", "ch1")

	_, _ = svc.ApplyRevisionImpact(context.Background(), "ch1", "RELISTEN_REQUIRED")

	p, _ := svc.GetProgress(context.Background(), "u1", "ch1")
	if p.CompletedAt == "" {
		t.Fatal("completion must be preserved, not silently reset")
	}
}
