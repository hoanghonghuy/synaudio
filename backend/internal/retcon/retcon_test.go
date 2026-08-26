package retcon

import (
	"context"
	"testing"
)

func TestCreateRetconRequest(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	r, err := svc.CreateRetconRequest(context.Background(), CreateRetconInput{
		StoryID:         "s1",
		TargetChapterID: "ch1",
		ProposedChange:  "Minh lost the key",
		Reason:          "Fix incorrect fact",
		RequestedBy:     "u1",
	})
	if err != nil {
		t.Fatalf("create retcon: %v", err)
	}
	if r.Status != "DRAFT" {
		t.Fatalf("expected DRAFT, got %q", r.Status)
	}
	if r.ImpactScope != "LOCAL" {
		t.Fatalf("expected LOCAL, got %q", r.ImpactScope)
	}
}

func TestCreateRetconRequestRejectsEmptyReason(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.CreateRetconRequest(context.Background(), CreateRetconInput{
		StoryID: "s1",
	}); err == nil {
		t.Fatal("expected error for empty reason")
	}
}

func TestApproveRetconRequest(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	r, _ := svc.CreateRetconRequest(context.Background(), CreateRetconInput{
		StoryID: "s1", TargetChapterID: "ch1", Reason: "fix", RequestedBy: "u1",
	})

	approved, err := svc.ApproveRetconRequest(context.Background(), r.ID, "u2")
	if err != nil {
		t.Fatalf("approve retcon: %v", err)
	}
	if approved.Status != "APPROVED" {
		t.Fatalf("expected APPROVED, got %q", approved.Status)
	}
	if approved.ApprovedBy != "u2" {
		t.Fatalf("expected approved_by u2, got %q", approved.ApprovedBy)
	}
}

func TestCancelRetconRequest(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	r, _ := svc.CreateRetconRequest(context.Background(), CreateRetconInput{
		StoryID: "s1", TargetChapterID: "ch1", Reason: "fix", RequestedBy: "u1",
	})

	cancelled, err := svc.CancelRetconRequest(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("cancel retcon: %v", err)
	}
	if cancelled.Status != "CANCELLED" {
		t.Fatalf("expected CANCELLED, got %q", cancelled.Status)
	}
}

func TestListRetconRequests(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	_, _ = svc.CreateRetconRequest(context.Background(), CreateRetconInput{
		StoryID: "s1", Reason: "fix1", RequestedBy: "u1",
	})
	_, _ = svc.CreateRetconRequest(context.Background(), CreateRetconInput{
		StoryID: "s1", Reason: "fix2", RequestedBy: "u1",
	})

	list, err := svc.ListRetconRequests(context.Background(), "s1")
	if err != nil {
		t.Fatalf("list retcons: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 retcons, got %d", len(list))
	}
}
