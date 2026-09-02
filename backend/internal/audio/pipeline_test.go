package audio

import (
	"context"
	"testing"
)

func TestSynthesizeNarrationProducesAudioAsset(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, WithTTS(NewMockTTS()), WithObjectStorage(newFakeObjectStorage()))

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "First sentence. Second sentence.", "u1")

	asset, err := svc.SynthesizeNarration(context.Background(), nar.ID)
	if err != nil {
		t.Fatalf("synthesize narration: %v", err)
	}
	if asset.ChapterID != "c1" {
		t.Fatalf("expected chapter c1, got %q", asset.ChapterID)
	}
	if asset.SourceNarrationRevisionID != nar.ID {
		t.Fatalf("expected source narration %q, got %q", nar.ID, asset.SourceNarrationRevisionID)
	}
	if asset.DurationMs <= 0 {
		t.Fatal("expected positive total duration")
	}
	if asset.StorageKey == "" {
		t.Fatal("expected storage key")
	}
}

func TestSynthesizeNarrationWithoutTTSFails(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, WithObjectStorage(newFakeObjectStorage()))

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")

	if _, err := svc.SynthesizeNarration(context.Background(), nar.ID); err == nil {
		t.Fatal("expected error when TTS not configured")
	}
}
