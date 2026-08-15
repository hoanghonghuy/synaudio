package generation

import (
	"context"
	"testing"
)

func TestWriteChapterCreatesContentRevision(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, WithTextAI(NewMockTextAI()))

	r, err := svc.WriteChapter(context.Background(), "c1", "Write chapter 1", "u1")
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if r.ContentText == "" {
		t.Fatal("expected non-empty content")
	}
	if r.SourceType != "AI_GENERATED" {
		t.Fatalf("expected AI_GENERATED, got %q", r.SourceType)
	}
	if r.Status != "CANDIDATE" {
		t.Fatalf("expected CANDIDATE, got %q", r.Status)
	}
}

func TestWriteChapterWithoutTextAIFails(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.WriteChapter(context.Background(), "c1", "prompt", "u1"); err == nil {
		t.Fatal("expected error when text AI not configured")
	}
}
