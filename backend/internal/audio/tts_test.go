package audio

import (
	"context"
	"testing"
	"time"
)

func TestCreateTTSSegmentsSplitsScript(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, WithTTS(NewMockTTS()))

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "First sentence. Second sentence. Third sentence.", "u1")

	segments, err := svc.CreateTTSSegments(context.Background(), nar.ID)
	if err != nil {
		t.Fatalf("create segments: %v", err)
	}
	if len(segments) != 3 {
		t.Fatalf("expected 3 segments, got %d", len(segments))
	}
	for i, seg := range segments {
		if seg.SegmentNo != i+1 {
			t.Fatalf("expected segment %d, got %d", i+1, seg.SegmentNo)
		}
	}
}

func TestSynthesizeSegmentProducesAudio(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, WithTTS(NewMockTTS()), WithObjectStorage(newFakeObjectStorage()))

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello world.", "u1")
	segments, _ := svc.CreateTTSSegments(context.Background(), nar.ID)

	synthesized, err := svc.SynthesizeSegment(context.Background(), segments[0].ID)
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if synthesized.Status != "SYNTHESIZED" {
		t.Fatalf("expected SYNTHESIZED, got %q", synthesized.Status)
	}
	if synthesized.DurationMs <= 0 {
		t.Fatal("expected positive duration")
	}
	if synthesized.TempStorageKey == "" {
		t.Fatal("expected temp storage key")
	}
}

func TestSynthesizeSegmentWithoutTTSFails(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, WithObjectStorage(newFakeObjectStorage()))

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello world.", "u1")
	segments, _ := svc.CreateTTSSegments(context.Background(), nar.ID)

	if _, err := svc.SynthesizeSegment(context.Background(), segments[0].ID); err == nil {
		t.Fatal("expected error when TTS not configured")
	}
}

type fakePresigner struct {
	url string
}

func (f fakePresigner) PresignedGetObject(_ context.Context, key string, _ time.Duration) (string, error) {
	return f.url + "/" + key, nil
}

func TestGetAudioURLReturnsPresignedURL(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, WithPresigner(fakePresigner{url: "https://cdn.example.com"}))

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")
	asset, _ := svc.CreateAudioAsset(context.Background(), "c1", nar.ID, "audio/c1/v1.mp3", "audio/mpeg", 100, 1000, 128)
	_, _ = svc.ActivateAudioAsset(context.Background(), "c1", asset.ID)

	url, err := svc.GetAudioURL(context.Background(), "c1")
	if err != nil {
		t.Fatalf("get audio url: %v", err)
	}
	if url != "https://cdn.example.com/audio/c1/v1.mp3" {
		t.Fatalf("unexpected url: %q", url)
	}
}

func TestGetAudioURLWithoutActiveAssetFails(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, WithPresigner(fakePresigner{url: "https://cdn.example.com"}))

	if _, err := svc.GetAudioURL(context.Background(), "c1"); err == nil {
		t.Fatal("expected error when no active asset")
	}
}
