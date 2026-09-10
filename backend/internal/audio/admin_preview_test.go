package audio

import (
	"context"
	"errors"
	"testing"
)

func TestGetAdminAudioPreviewURLAllowsReadyAssetBeforePublication(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(store, WithPresigner(fakePresigner{url: "https://cdn.example.com"}))

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")
	asset, err := svc.CreateAudioAsset(context.Background(), "c1", nar.ID, "audio/c1/v1.mp3", "audio/mpeg", 100, 1000, 128)
	if err != nil {
		t.Fatalf("create audio asset: %v", err)
	}

	url, err := svc.GetAdminAudioPreviewURL(context.Background(), "c1", asset.ID)
	if err != nil {
		t.Fatalf("get admin preview url: %v", err)
	}
	if url != "https://cdn.example.com/audio/c1/v1.mp3" {
		t.Fatalf("unexpected url: %q", url)
	}
}

func TestGetAdminAudioPreviewURLRejectsCrossChapterAsset(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(store, WithPresigner(fakePresigner{url: "https://cdn.example.com"}))

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")
	asset, _ := svc.CreateAudioAsset(context.Background(), "c1", nar.ID, "audio/c1/v1.mp3", "audio/mpeg", 100, 1000, 128)

	if _, err := svc.GetAdminAudioPreviewURL(context.Background(), "c2", asset.ID); !errors.Is(err, ErrAudioAssetNotFound) {
		t.Fatalf("expected concealed not-found for cross-chapter asset, got %v", err)
	}
}

func TestGetAdminAudioPreviewURLRejectsMissingAsset(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(store, WithPresigner(fakePresigner{url: "https://cdn.example.com"}))

	if _, err := svc.GetAdminAudioPreviewURL(context.Background(), "c1", "missing"); !errors.Is(err, ErrAudioAssetNotFound) {
		t.Fatalf("expected ErrAudioAssetNotFound, got %v", err)
	}
}

func TestGetAdminAudioPreviewURLRejectsNonReadyAsset(t *testing.T) {
	store := newFakeStore()
	store.assets["c1"] = []AudioAsset{{
		ID:         "a1",
		ChapterID:  "c1",
		Status:     "PROCESSING",
		StorageKey: "audio/c1/incomplete.mp3",
	}}
	svc := newTestService(store, WithPresigner(fakePresigner{url: "https://cdn.example.com"}))

	if _, err := svc.GetAdminAudioPreviewURL(context.Background(), "c1", "a1"); !errors.Is(err, ErrAdminAudioPreviewNotEligible) {
		t.Fatalf("expected ErrAdminAudioPreviewNotEligible, got %v", err)
	}
}

func TestGetAdminAudioPreviewURLRequiresPresigner(t *testing.T) {
	store := newFakeStore()
	store.assets["c1"] = []AudioAsset{{
		ID:         "a1",
		ChapterID:  "c1",
		Status:     "READY",
		StorageKey: "audio/c1/v1.mp3",
	}}
	svc := newTestService(store)

	if _, err := svc.GetAdminAudioPreviewURL(context.Background(), "c1", "a1"); !errors.Is(err, ErrAudioPresignerRequired) {
		t.Fatalf("expected ErrAudioPresignerRequired, got %v", err)
	}
}
