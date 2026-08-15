package generation

import (
	"context"
	"testing"
)

func TestRewriteChapterCreatesNewRevision(t *testing.T) {
	svc := NewService(newFakeStore(), WithTextAI(NewMockTextAI()))

	// Create an initial revision.
	orig, err := svc.CreateContentRevision(context.Background(), "c1", "original text", "AI_GENERATED", "u1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	rewritten, err := svc.RewriteChapter(context.Background(), "c1", orig.ID, "make it better", "u1")
	if err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if rewritten.ID == orig.ID {
		t.Fatal("expected a new revision ID")
	}
	if rewritten.RevisionNo <= orig.RevisionNo {
		t.Fatalf("expected higher revision number, got %d vs %d", rewritten.RevisionNo, orig.RevisionNo)
	}
	if rewritten.SourceType != "AI_REWRITE" {
		t.Fatalf("expected AI_REWRITE, got %q", rewritten.SourceType)
	}
	if rewritten.BasedOnRevisionID != orig.ID {
		t.Fatalf("expected based_on %q, got %q", orig.ID, rewritten.BasedOnRevisionID)
	}
}

func TestRewriteChapterWithoutTextAIFails(t *testing.T) {
	svc := NewService(newFakeStore())

	if _, err := svc.RewriteChapter(context.Background(), "c1", "rev1", "feedback", "u1"); err == nil {
		t.Fatal("expected error when text AI not configured")
	}
}
