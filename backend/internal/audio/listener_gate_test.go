package audio

import (
	"context"
	"errors"
	"testing"
)

func TestGetListenerAudioURLRejectsIneligibleChapter(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(
		store,
		WithPresigner(fakePresigner{url: "https://cdn.example.com"}),
		WithListenerAudioGate(&fakeListenerAudioGate{err: ErrListenerAudioNotEligible}),
	)

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")
	asset, _ := svc.CreateAudioAsset(context.Background(), "c1", nar.ID, "audio/c1/v1.mp3", "audio/mpeg", 100, 1000, 128)
	_, _ = svc.ActivateAudioAsset(context.Background(), "c1", asset.ID)

	if _, err := svc.GetListenerAudioURL(context.Background(), "c1"); !errors.Is(err, ErrListenerAudioNotEligible) {
		t.Fatalf("expected ErrListenerAudioNotEligible, got %v", err)
	}
}

func TestGetListenerAudioURLFailsClosedWithoutGate(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(store, WithPresigner(fakePresigner{url: "https://cdn.example.com"}))

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")
	asset, _ := svc.CreateAudioAsset(context.Background(), "c1", nar.ID, "audio/c1/v1.mp3", "audio/mpeg", 100, 1000, 128)
	_, _ = svc.ActivateAudioAsset(context.Background(), "c1", asset.ID)

	if _, err := svc.GetListenerAudioURL(context.Background(), "c1"); !errors.Is(err, ErrListenerAudioGateRequired) {
		t.Fatalf("expected ErrListenerAudioGateRequired, got %v", err)
	}
}

func TestGetListenerAudioURLReturnsPresignedURLForEligibleChapter(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(
		store,
		WithPresigner(fakePresigner{url: "https://cdn.example.com"}),
		WithListenerAudioGate(&fakeListenerAudioGate{}),
	)

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")
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
