package audio

import (
	"context"
	"errors"
	"testing"
)

func TestGetLatestNarrationRevisionReturnsNewestRevision(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	store.narrations["c1"] = []NarrationRevision{
		{ID: "nar-1", ChapterID: "c1", RevisionNo: 1, Script: "one"},
		{ID: "nar-2", ChapterID: "c1", RevisionNo: 2, Script: "two"},
	}

	got, err := svc.GetLatestNarrationRevision(context.Background(), "c1")
	if err != nil {
		t.Fatalf("get latest narration: %v", err)
	}
	if got.ID != "nar-2" {
		t.Fatalf("expected nar-2, got %q", got.ID)
	}
}

func TestGetLatestNarrationRevisionReturnsNotFoundForEmptyChapter(t *testing.T) {
	svc := NewService(newFakeStore())
	if _, err := svc.GetLatestNarrationRevision(context.Background(), "c1"); !errors.Is(err, ErrNarrationNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestGetLatestReadyAudioAssetReturnsNewestNonActiveReady(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	store.assets["c1"] = []AudioAsset{
		{ID: "a1", ChapterID: "c1", VersionNo: 1, Status: "READY", IsActive: true},
		{ID: "a2", ChapterID: "c1", VersionNo: 2, Status: "READY", IsActive: false},
		{ID: "a3", ChapterID: "c1", VersionNo: 3, Status: "FAILED", IsActive: false},
	}

	got, err := svc.GetLatestReadyAudioAsset(context.Background(), "c1")
	if err != nil {
		t.Fatalf("get latest ready audio: %v", err)
	}
	if got.ID != "a2" {
		t.Fatalf("expected a2, got %q", got.ID)
	}
}

func TestGetLatestReadyAudioAssetReturnsNotFoundWhenOnlyActiveExists(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	store.assets["c1"] = []AudioAsset{
		{ID: "a1", ChapterID: "c1", VersionNo: 1, Status: "READY", IsActive: true},
	}

	if _, err := svc.GetLatestReadyAudioAsset(context.Background(), "c1"); !errors.Is(err, ErrReadyAudioAssetNotFound) {
		t.Fatalf("expected ready-not-found, got %v", err)
	}
}

func TestSynthesizeNarrationForChapterRejectsOwnershipMismatch(t *testing.T) {
	store := newFakeStore()
	svc := NewService(
		store,
		WithTTS(NewMockTTS()),
		WithObjectStorage(newFakeObjectStorage()),
		WithAudioProcessor(NewMockAudioProcessor()),
	)

	nar, err := svc.CreateNarrationRevision(context.Background(), "chapter-a", "cr-1", "voice-1", "Hello.", "u1")
	if err != nil {
		t.Fatalf("create narration: %v", err)
	}

	if _, err := svc.SynthesizeNarrationForChapter(context.Background(), "chapter-b", nar.ID); !errors.Is(err, ErrNarrationChapterMismatch) {
		t.Fatalf("expected chapter mismatch, got %v", err)
	}
}
