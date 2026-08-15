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
}

func TestApproveContentRejectsMissingRevision(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.ApproveContent(context.Background(), "c1", "missing", "admin1"); !errors.Is(err, ErrContentRevisionNotFound) {
		t.Fatalf("expected ErrContentRevisionNotFound, got %v", err)
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
