package audio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"testing"
)

func TestComputeAndVerifyAudioFileChecksum(t *testing.T) {
	path := t.TempDir() + "/final.mp3"
	data := []byte("exact-final-media")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	expected := "sha256:" + hex.EncodeToString(sum[:])

	got, err := ComputeAudioFileChecksum(context.Background(), path)
	if err != nil {
		t.Fatalf("compute checksum: %v", err)
	}
	if got != expected {
		t.Fatalf("checksum=%q want %q", got, expected)
	}
	if err := VerifyAudioFileChecksum(context.Background(), path, expected); err != nil {
		t.Fatalf("verify checksum: %v", err)
	}
}

func TestVerifyAudioFileChecksumFailsHonest(t *testing.T) {
	path := t.TempDir() + "/final.mp3"
	if err := os.WriteFile(path, []byte("media"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyAudioFileChecksum(context.Background(), path, ""); !errors.Is(err, ErrChecksumMissing) {
		t.Fatalf("missing checksum err=%v", err)
	}
	wrong := "sha256:" + string(make([]byte, sha256.Size*2))
	if err := VerifyAudioFileChecksum(context.Background(), path, wrong); err == nil {
		t.Fatal("expected malformed checksum to fail")
	}
	sum := sha256.Sum256([]byte("different-media"))
	mismatch := "sha256:" + hex.EncodeToString(sum[:])
	if err := VerifyAudioFileChecksum(context.Background(), path, mismatch); !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("mismatch err=%v", err)
	}
}

func TestSynthesizeNarrationPersistsChecksumForExactFinalBytes(t *testing.T) {
	objects := newFakeObjectStorage()
	store := newFakeStore()
	processor := &recordingProcessor{output: []byte("FINAL-AUDIO")}
	svc := NewService(store,
		WithTTS(NewMockTTS()),
		WithObjectStorage(objects),
		WithAudioProcessor(processor),
	)
	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")
	asset, err := svc.SynthesizeNarration(context.Background(), nar.ID)
	if err != nil {
		t.Fatalf("synthesize narration: %v", err)
	}
	final := objects.objects[asset.StorageKey]
	sum := sha256.Sum256(final)
	want := "sha256:" + hex.EncodeToString(sum[:])
	if asset.Checksum != want {
		t.Fatalf("checksum=%q want %q", asset.Checksum, want)
	}
}

type vanishingProcessor struct{ *recordingProcessor }

func (p *vanishingProcessor) ProcessFiles(ctx context.Context, inputPaths []string, outputPath string) (int, error) {
	duration, err := p.recordingProcessor.ProcessFiles(ctx, inputPaths, outputPath)
	if err != nil {
		return 0, err
	}
	if err := os.Remove(outputPath); err != nil {
		return 0, err
	}
	return duration, nil
}

func TestChecksumFailureCannotCreateReadyAsset(t *testing.T) {
	objects := newFakeObjectStorage()
	store := newFakeStore()
	processor := &vanishingProcessor{recordingProcessor: &recordingProcessor{output: []byte("FINAL-AUDIO")}}
	svc := NewService(store,
		WithTTS(NewMockTTS()),
		WithObjectStorage(objects),
		WithAudioProcessor(processor),
	)
	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")
	if _, err := svc.SynthesizeNarration(context.Background(), nar.ID); err == nil {
		t.Fatal("expected checksum failure")
	}
	if len(store.assets["c1"]) != 0 {
		t.Fatal("checksum failure must not create READY metadata")
	}
	for key := range objects.objects {
		if len(key) >= len("chapters/") && key[:len("chapters/")] == "chapters/" {
			t.Fatalf("checksum failure must not upload final object %q", key)
		}
	}
}
