package generation

import (
	"context"
	"errors"
	"testing"
)

func TestCreateContentRevisionAssignsSequentialRevision(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	r1, err := svc.CreateContentRevision(context.Background(), "c1", "Once upon a time...", "AI_GENERATED", "u1")
	if err != nil {
		t.Fatalf("create revision 1: %v", err)
	}
	if r1.RevisionNo != 1 {
		t.Fatalf("expected revision 1, got %d", r1.RevisionNo)
	}
	if r1.Status != "CANDIDATE" {
		t.Fatalf("expected CANDIDATE, got %q", r1.Status)
	}

	r2, err := svc.CreateContentRevision(context.Background(), "c1", "Once upon a time, again...", "AI_REWRITE", "u1")
	if err != nil {
		t.Fatalf("create revision 2: %v", err)
	}
	if r2.RevisionNo != 2 {
		t.Fatalf("expected revision 2, got %d", r2.RevisionNo)
	}
}

func TestCreateContentRevisionRejectsEmptyText(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.CreateContentRevision(context.Background(), "c1", "  ", "AI_GENERATED", "u1"); err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestApproveContentMarksRevisionApproved(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	r, _ := svc.CreateContentRevision(context.Background(), "c1", "Once upon a time...", "AI_GENERATED", "u1")

	a, err := svc.ApproveContent(context.Background(), "c1", r.ID, "admin1")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if a.ContentRevisionID != r.ID {
		t.Fatalf("expected revision %q, got %q", r.ID, a.ContentRevisionID)
	}
	persisted, err := store.GetContentRevision(context.Background(), r.ID)
	if err != nil {
		t.Fatalf("read approved revision: %v", err)
	}
	if persisted.Status != "APPROVED" {
		t.Fatalf("expected APPROVED projection, got %q", persisted.Status)
	}
}

func TestApproveContentRejectsMissingRevision(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.ApproveContent(context.Background(), "c1", "missing", "admin1"); !errors.Is(err, ErrContentRevisionNotFound) {
		t.Fatalf("expected ErrContentRevisionNotFound, got %v", err)
	}
}

func TestApproveContentRejectsCrossChapterRevision(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	r, _ := svc.CreateContentRevision(context.Background(), "c1", "chapter one", "AI_GENERATED", "u1")

	if _, err := svc.ApproveContent(context.Background(), "c2", r.ID, "admin1"); !errors.Is(err, ErrContentRevisionChapterMismatch) {
		t.Fatalf("expected ErrContentRevisionChapterMismatch, got %v", err)
	}
	persisted, _ := store.GetContentRevision(context.Background(), r.ID)
	if persisted.Status != "CANDIDATE" {
		t.Fatalf("cross-chapter approval mutated revision: %q", persisted.Status)
	}
	if len(store.approvals["c2"]) != 0 {
		t.Fatalf("cross-chapter approval created side effect: %#v", store.approvals["c2"])
	}
}

func TestApproveContentRetryIsIdempotent(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	r, _ := svc.CreateContentRevision(context.Background(), "c1", "chapter one", "AI_GENERATED", "u1")

	first, err := svc.ApproveContent(context.Background(), "c1", r.ID, "admin1")
	if err != nil {
		t.Fatalf("first approve: %v", err)
	}
	second, err := svc.ApproveContent(context.Background(), "c1", r.ID, "admin1")
	if err != nil {
		t.Fatalf("retry approve: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("retry created a new approval: first=%q second=%q", first.ID, second.ID)
	}
	if len(store.approvals["c1"]) != 1 {
		t.Fatalf("expected one approval side effect, got %d", len(store.approvals["c1"]))
	}
}

func TestApproveContentRejectsTerminalRevision(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	r, _ := svc.CreateContentRevision(context.Background(), "c1", "chapter one", "AI_GENERATED", "u1")
	_, _ = store.UpdateContentRevisionStatus(context.Background(), r.ID, "REJECTED")

	if _, err := svc.ApproveContent(context.Background(), "c1", r.ID, "admin1"); !errors.Is(err, ErrContentRevisionNotApprovable) {
		t.Fatalf("expected ErrContentRevisionNotApprovable, got %v", err)
	}
	if len(store.approvals["c1"]) != 0 {
		t.Fatalf("terminal revision approval created side effect: %#v", store.approvals["c1"])
	}
}

func TestListContentRevisionsReturnsAll(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	_, _ = svc.CreateContentRevision(context.Background(), "c1", "text one", "AI_GENERATED", "u1")
	_, _ = svc.CreateContentRevision(context.Background(), "c1", "text two", "AI_REWRITE", "u1")

	revisions, err := svc.ListContentRevisions(context.Background(), "c1")
	if err != nil {
		t.Fatalf("list revisions: %v", err)
	}
	if len(revisions) != 2 {
		t.Fatalf("expected 2 revisions, got %d", len(revisions))
	}
}
