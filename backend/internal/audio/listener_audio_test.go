package audio

import (
	"context"
	"errors"
	"testing"
)

func TestGetListenerAudioURLRejectsStaleActiveAudioAfterNewerNarration(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(
		store,
		WithPresigner(fakePresigner{url: "https://cdn.example.com"}),
		WithListenerAudioGate(&fakeListenerAudioGate{}),
	)

	store.narrations["c1"] = []NarrationRevision{
		{ID: "nar-1", ChapterID: "c1", RevisionNo: 1, SourceContentRevisionID: "cr-1"},
		{ID: "nar-2", ChapterID: "c1", RevisionNo: 2, SourceContentRevisionID: "cr-1"},
	}
	store.assets["c1"] = []AudioAsset{
		{ID: "asset-1", ChapterID: "c1", VersionNo: 1, SourceNarrationRevisionID: "nar-1", Status: "READY", IsActive: true},
	}

	if _, err := svc.GetListenerAudioURL(context.Background(), "c1"); !errors.Is(err, ErrListenerAudioNotEligible) {
		t.Fatalf("expected ErrListenerAudioNotEligible for stale active audio, got %v", err)
	}
}

func TestGetListenerAudioURLReturnsPresignedURLWhenActiveAudioMatchesLatestNarration(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(
		store,
		WithPresigner(fakePresigner{url: "https://cdn.example.com"}),
		WithListenerAudioGate(&fakeListenerAudioGate{}),
	)

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr-1", "voice-1", "Hello.", "u1")
	asset, _ := svc.CreateAudioAsset(context.Background(), "c1", nar.ID, "audio/c1/v1.mp3", "audio/mpeg", 100, 1000, 128)
	_, _ = svc.ActivateAudioAsset(context.Background(), "c1", asset.ID)

	url, err := svc.GetListenerAudioURL(context.Background(), "c1")
	if err != nil {
		t.Fatalf("get listener audio url: %v", err)
	}
	if url != "https://cdn.example.com/audio/c1/v1.mp3" {
		t.Fatalf("unexpected url: %q", url)
	}
}
