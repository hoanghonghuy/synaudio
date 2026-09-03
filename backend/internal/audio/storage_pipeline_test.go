package audio

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeObjectStorage struct {
	objects map[string][]byte
	putErr  error
	getErr  error
}

func newFakeObjectStorage() *fakeObjectStorage {
	return &fakeObjectStorage{objects: map[string][]byte{}}
}

func (s *fakeObjectStorage) Put(_ context.Context, key string, data []byte) error {
	if s.putErr != nil {
		return s.putErr
	}
	s.objects[key] = append([]byte(nil), data...)
	return nil
}

func (s *fakeObjectStorage) Get(_ context.Context, key string) ([]byte, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	data, ok := s.objects[key]
	if !ok {
		return nil, errors.New("object not found")
	}
	return append([]byte(nil), data...), nil
}

type recordingProcessor struct {
	segments [][]byte
	output   []byte
	err      error
}

func (p *recordingProcessor) Process(_ context.Context, in ProcessInput) (ProcessOutput, error) {
	if p.err != nil {
		return ProcessOutput{}, p.err
	}
	p.segments = make([][]byte, len(in.Segments))
	total := 0
	for i, seg := range in.Segments {
		p.segments[i] = append([]byte(nil), seg.Data...)
		total += seg.DurationMs
	}
	return ProcessOutput{Data: append([]byte(nil), p.output...), DurationMs: total}, nil
}

type storageAwareStore struct {
	*fakeStore
	storage *fakeObjectStorage
}

func (s *storageAwareStore) CreateAudioAsset(ctx context.Context, a AudioAsset) (AudioAsset, error) {
	if _, ok := s.storage.objects[a.StorageKey]; !ok {
		return AudioAsset{}, errors.New("final object missing before READY metadata")
	}
	return s.fakeStore.CreateAudioAsset(ctx, a)
}

func TestSynthesizeSegmentPersistsAudioBeforeSuccess(t *testing.T) {
	store := newFakeStore()
	objects := newFakeObjectStorage()
	svc := NewService(store, WithTTS(NewMockTTS()), WithObjectStorage(objects))

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello world.", "u1")
	segments, _ := svc.CreateTTSSegments(context.Background(), nar.ID)

	synthesized, err := svc.SynthesizeSegment(context.Background(), segments[0].ID)
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if got := objects.objects[synthesized.TempStorageKey]; !bytes.Equal(got, []byte("MOCK-AUDIO")) {
		t.Fatalf("expected persisted TTS bytes, got %q", got)
	}
}

func TestSynthesizeSegmentStorageFailureDoesNotMarkSuccess(t *testing.T) {
	store := newFakeStore()
	objects := newFakeObjectStorage()
	objects.putErr = errors.New("storage unavailable")
	svc := NewService(store, WithTTS(NewMockTTS()), WithObjectStorage(objects))

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello world.", "u1")
	segments, _ := svc.CreateTTSSegments(context.Background(), nar.ID)

	if _, err := svc.SynthesizeSegment(context.Background(), segments[0].ID); err == nil {
		t.Fatal("expected storage error")
	}
	persisted, err := store.GetTTSSegment(context.Background(), segments[0].ID)
	if err != nil {
		t.Fatalf("get segment: %v", err)
	}
	if persisted.Status == "SYNTHESIZED" {
		t.Fatal("segment must not be marked SYNTHESIZED after storage failure")
	}
}

func TestSynthesizeNarrationWithoutProcessorFailsClosed(t *testing.T) {
	objects := newFakeObjectStorage()
	store := newFakeStore()
	svc := NewService(store, WithTTS(NewMockTTS()), WithObjectStorage(objects))

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")
	_, err := svc.SynthesizeNarration(context.Background(), nar.ID)
	if err == nil || !strings.Contains(err.Error(), "audio processor not configured") {
		t.Fatalf("expected fail-closed processor error, got %v", err)
	}
	if len(store.assets["c1"]) != 0 {
		t.Fatal("READY asset must not be created without an explicitly configured processor")
	}
	if len(objects.objects) != 0 {
		t.Fatal("pipeline must reject missing processor before staging or final audio side effects")
	}
}

func TestSynthesizeNarrationUsesStagedBytesAndPersistsFinalBeforeReady(t *testing.T) {
	objects := newFakeObjectStorage()
	store := &storageAwareStore{fakeStore: newFakeStore(), storage: objects}
	processor := &recordingProcessor{output: []byte("FINAL-AUDIO")}
	svc := NewService(store,
		WithTTS(NewMockTTS()),
		WithObjectStorage(objects),
		WithAudioProcessor(processor),
	)

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "First sentence. Second sentence.", "u1")
	asset, err := svc.SynthesizeNarration(context.Background(), nar.ID)
	if err != nil {
		t.Fatalf("synthesize narration: %v", err)
	}
	if len(processor.segments) != 2 {
		t.Fatalf("expected 2 processor segments, got %d", len(processor.segments))
	}
	for i, got := range processor.segments {
		if !bytes.Equal(got, []byte("MOCK-AUDIO")) {
			t.Fatalf("processor segment %d received %q instead of media bytes", i, got)
		}
	}
	if got := objects.objects[asset.StorageKey]; !bytes.Equal(got, []byte("FINAL-AUDIO")) {
		t.Fatalf("expected final bytes persisted before READY metadata, got %q", got)
	}
}

func TestSynthesizeNarrationProcessorFailurePreventsReadyAsset(t *testing.T) {
	objects := newFakeObjectStorage()
	store := newFakeStore()
	processor := &recordingProcessor{err: errors.New("ffmpeg execution failed")}
	svc := NewService(store,
		WithTTS(NewMockTTS()),
		WithObjectStorage(objects),
		WithAudioProcessor(processor),
	)

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")
	if _, err := svc.SynthesizeNarration(context.Background(), nar.ID); err == nil || !strings.Contains(err.Error(), "process audio") {
		t.Fatalf("expected processor failure, got %v", err)
	}
	if len(store.assets["c1"]) != 0 {
		t.Fatal("READY asset must not be created when processor fails")
	}
	for key := range objects.objects {
		if strings.Contains(key, "/audio/") {
			t.Fatalf("final audio object %q must not exist after processor failure", key)
		}
	}
}

func TestSynthesizeNarrationFinalStorageFailurePreventsReadyAsset(t *testing.T) {
	objects := newFakeObjectStorage()
	store := newFakeStore()
	processor := &recordingProcessor{output: []byte("FINAL-AUDIO")}
	svc := NewService(store,
		WithTTS(NewMockTTS()),
		WithObjectStorage(objects),
		WithAudioProcessor(processor),
	)

	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")
	// Allow staging Put, then fail the final Put after the processor has run.
	objects.putErr = nil
	processor.output = []byte("FINAL-AUDIO")

	// Use a storage implementation that fails only for final chapter keys.
	conditional := &conditionalFailStorage{base: objects}
	svc = NewService(store,
		WithTTS(NewMockTTS()),
		WithObjectStorage(conditional),
		WithAudioProcessor(processor),
	)
	if _, err := svc.SynthesizeNarration(context.Background(), nar.ID); err == nil {
		t.Fatal("expected final storage error")
	}
	if len(store.assets["c1"]) != 0 {
		t.Fatal("READY asset must not be created when final storage fails")
	}
}

type conditionalFailStorage struct {
	base *fakeObjectStorage
}

func (s *conditionalFailStorage) Put(ctx context.Context, key string, data []byte) error {
	if len(key) >= len("chapters/") && key[:len("chapters/")] == "chapters/" {
		return errors.New("final storage unavailable")
	}
	return s.base.Put(ctx, key, data)
}

func (s *conditionalFailStorage) Get(ctx context.Context, key string) ([]byte, error) {
	return s.base.Get(ctx, key)
}
