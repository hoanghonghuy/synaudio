package retcon

import (
	"context"
	"errors"
	"testing"
)

func TestAnalyzeRetconRequestMovesToAnalyzing(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	r, _ := svc.CreateRetconRequest(context.Background(), CreateRetconInput{
		StoryID: "s1", TargetChapterID: "ch1", Reason: "fix", RequestedBy: "u1",
	})

	analyzed, err := svc.AnalyzeRetconRequest(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("analyze retcon: %v", err)
	}
	if analyzed.Status != "ANALYZING" {
		t.Fatalf("expected ANALYZING, got %q", analyzed.Status)
	}
}

func TestMarkReadyToApplyRequiresApproved(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	r, _ := svc.CreateRetconRequest(context.Background(), CreateRetconInput{
		StoryID: "s1", TargetChapterID: "ch1", Reason: "fix", RequestedBy: "u1",
	})

	// Not approved yet, so cannot be marked ready.
	if _, err := svc.MarkReadyToApply(context.Background(), r.ID); err == nil {
		t.Fatal("expected error when marking ready before approval")
	}
}

func TestMarkReadyToApplyAfterApproval(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	r, _ := svc.CreateRetconRequest(context.Background(), CreateRetconInput{
		StoryID: "s1", TargetChapterID: "ch1", Reason: "fix", RequestedBy: "u1",
	})
	_, _ = svc.ApproveRetconRequest(context.Background(), r.ID, "u2")

	ready, err := svc.MarkReadyToApply(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("mark ready: %v", err)
	}
	if ready.Status != "READY_TO_APPLY" {
		t.Fatalf("expected READY_TO_APPLY, got %q", ready.Status)
	}
}

func TestApplyRetconRequestRequiresReady(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	r, _ := svc.CreateRetconRequest(context.Background(), CreateRetconInput{
		StoryID: "s1", TargetChapterID: "ch1", Reason: "fix", RequestedBy: "u1",
	})

	// DRAFT is not READY_TO_APPLY.
	if _, err := svc.ApplyRetconRequest(context.Background(), r.ID, "u3"); !errors.Is(err, ErrRetconNotReady) {
		t.Fatalf("expected ErrRetconNotReady, got %v", err)
	}
}

func TestApplyRetconRequestSucceeds(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	r, _ := svc.CreateRetconRequest(context.Background(), CreateRetconInput{
		StoryID: "s1", TargetChapterID: "ch1", Reason: "fix", RequestedBy: "u1",
	})
	_, _ = svc.ApproveRetconRequest(context.Background(), r.ID, "u2")
	_, _ = svc.MarkReadyToApply(context.Background(), r.ID)

	applied, err := svc.ApplyRetconRequest(context.Background(), r.ID, "u3")
	if err != nil {
		t.Fatalf("apply retcon: %v", err)
	}
	if applied.Status != "APPLIED" {
		t.Fatalf("expected APPLIED, got %q", applied.Status)
	}
	if applied.AppliedBy != "u3" {
		t.Fatalf("expected applied_by u3, got %q", applied.AppliedBy)
	}
}
