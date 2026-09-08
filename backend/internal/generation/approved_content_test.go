package generation

import (
	"context"
	"errors"
	"testing"
)

func TestRequireApprovedContentRevisionAcceptsApprovedRevision(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	r, _ := svc.CreateContentRevision(context.Background(), "c1", "approved text", "AI_GENERATED", "u1")
	_, _ = svc.ApproveContent(context.Background(), "c1", r.ID, "admin1")

	if err := svc.RequireApprovedContentRevision(context.Background(), "c1", r.ID); err != nil {
		t.Fatalf("expected approved revision to pass, got %v", err)
	}
}

func TestRequireApprovedContentRevisionRejectsMissingRevision(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if err := svc.RequireApprovedContentRevision(context.Background(), "c1", "missing"); !errors.Is(err, ErrContentRevisionNotFound) {
		t.Fatalf("expected ErrContentRevisionNotFound, got %v", err)
	}
}

func TestRequireApprovedContentRevisionRejectsCrossChapterRevision(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	r, _ := svc.CreateContentRevision(context.Background(), "c1", "chapter one", "AI_GENERATED", "u1")
	_, _ = svc.ApproveContent(context.Background(), "c1", r.ID, "admin1")

	if err := svc.RequireApprovedContentRevision(context.Background(), "c2", r.ID); !errors.Is(err, ErrContentRevisionChapterMismatch) {
		t.Fatalf("expected ErrContentRevisionChapterMismatch, got %v", err)
	}
}

func TestRequireApprovedContentRevisionRejectsCandidateRevision(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	r, _ := svc.CreateContentRevision(context.Background(), "c1", "candidate text", "AI_GENERATED", "u1")

	if err := svc.RequireApprovedContentRevision(context.Background(), "c1", r.ID); !errors.Is(err, ErrContentRevisionNotApproved) {
		t.Fatalf("expected ErrContentRevisionNotApproved, got %v", err)
	}
}

func TestRequireApprovedContentRevisionRejectsRejectedRevision(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	r, _ := svc.CreateContentRevision(context.Background(), "c1", "rejected text", "AI_GENERATED", "u1")
	_, _ = store.UpdateContentRevisionStatus(context.Background(), r.ID, "REJECTED")

	if err := svc.RequireApprovedContentRevision(context.Background(), "c1", r.ID); !errors.Is(err, ErrContentRevisionNotApproved) {
		t.Fatalf("expected ErrContentRevisionNotApproved, got %v", err)
	}
}
