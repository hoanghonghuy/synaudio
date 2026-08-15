package generation

import (
	"context"
	"testing"
)

func TestEditContentCreatesAdminEditRevision(t *testing.T) {
	svc := NewService(newFakeStore())

	orig, _ := svc.CreateContentRevision(context.Background(), "c1", "original", "AI_GENERATED", "u1")

	edited, err := svc.EditContent(context.Background(), "c1", orig.ID, "edited text", "admin1")
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if edited.ID == orig.ID {
		t.Fatal("expected new revision ID")
	}
	if edited.SourceType != "ADMIN_EDIT" {
		t.Fatalf("expected ADMIN_EDIT, got %q", edited.SourceType)
	}
	if edited.BasedOnRevisionID != orig.ID {
		t.Fatalf("expected based_on %q, got %q", orig.ID, edited.BasedOnRevisionID)
	}
}

func TestEditContentRejectsEmptyText(t *testing.T) {
	svc := NewService(newFakeStore())

	orig, _ := svc.CreateContentRevision(context.Background(), "c1", "original", "AI_GENERATED", "u1")

	if _, err := svc.EditContent(context.Background(), "c1", orig.ID, "   ", "admin1"); err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestRegenerateContentCreatesNewCandidate(t *testing.T) {
	svc := NewService(newFakeStore(), WithTextAI(NewMockTextAI()))

	orig, _ := svc.CreateContentRevision(context.Background(), "c1", "original", "AI_GENERATED", "u1")

	regenerated, err := svc.RegenerateContent(context.Background(), "c1", orig.ID, "u1")
	if err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	if regenerated.ID == orig.ID {
		t.Fatal("expected new revision ID")
	}
	if regenerated.SourceType != "AI_GENERATED" {
		t.Fatalf("expected AI_GENERATED, got %q", regenerated.SourceType)
	}
}

func TestRejectContentMarksRevisionRejected(t *testing.T) {
	svc := NewService(newFakeStore())

	orig, _ := svc.CreateContentRevision(context.Background(), "c1", "original", "AI_GENERATED", "u1")

	rejected, err := svc.RejectContent(context.Background(), orig.ID, "admin1", "not good enough")
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if rejected.Status != "REJECTED" {
		t.Fatalf("expected REJECTED, got %q", rejected.Status)
	}
}
