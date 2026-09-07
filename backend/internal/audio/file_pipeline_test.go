package audio

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func (s *fakeObjectStorage) DownloadToFile(ctx context.Context, key, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := s.Get(ctx, key)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (s *fakeObjectStorage) UploadFile(ctx context.Context, key, path string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if err := s.Put(ctx, key, data); err != nil {
		return 0, err
	}
	return int64(len(data)), nil
}

func (s *conditionalFailStorage) DownloadToFile(ctx context.Context, key, path string) error {
	return s.base.DownloadToFile(ctx, key, path)
}

func (s *conditionalFailStorage) UploadFile(ctx context.Context, key, path string) (int64, error) {
	if strings.HasPrefix(key, "chapters/") {
		return 0, errors.New("final storage unavailable")
	}
	return s.base.UploadFile(ctx, key, path)
}

func (p *recordingProcessor) ProcessFiles(ctx context.Context, inputPaths []string, outputPath string) (int, error) {
	if p.err != nil {
		return 0, p.err
	}
	p.segments = make([][]byte, len(inputPaths))
	for i, path := range inputPaths {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return 0, err
		}
		p.segments[i] = append([]byte(nil), data...)
	}
	if err := os.WriteFile(outputPath, p.output, 0o600); err != nil {
		return 0, err
	}
	return len(inputPaths) * 400, nil
}

type cancellationFileStorage struct {
	*fakeObjectStorage
	downloads int
	cancel    context.CancelFunc
}

func (s *cancellationFileStorage) DownloadToFile(ctx context.Context, key, path string) error {
	s.downloads++
	if s.downloads == 1 {
		if err := s.fakeObjectStorage.DownloadToFile(ctx, key, path); err != nil {
			return err
		}
		s.cancel()
		return nil
	}
	return ctx.Err()
}

func (s *cancellationFileStorage) UploadFile(ctx context.Context, key, path string) (int64, error) {
	return s.fakeObjectStorage.UploadFile(ctx, key, path)
}

func TestSynthesizeNarrationFileBoundaryRequired(t *testing.T) {
	store := newFakeStore()
	objects := &byteOnlyStorage{base: newFakeObjectStorage()}
	svc := NewService(store,
		WithTTS(NewMockTTS()),
		WithObjectStorage(objects),
		WithAudioProcessor(NewMockAudioProcessor()),
	)
	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")
	_, err := svc.SynthesizeNarration(context.Background(), nar.ID)
	if err == nil || !strings.Contains(err.Error(), "file boundary") {
		t.Fatalf("expected file-boundary failure, got %v", err)
	}
	if len(objects.base.objects) != 0 {
		t.Fatal("must fail before creating TTS/final storage side effects")
	}
}

type byteOnlyStorage struct{ base *fakeObjectStorage }

func (s *byteOnlyStorage) Put(ctx context.Context, key string, data []byte) error {
	return s.base.Put(ctx, key, data)
}
func (s *byteOnlyStorage) Get(ctx context.Context, key string) ([]byte, error) {
	return s.base.Get(ctx, key)
}

func TestSynthesizeNarrationCancellationLeavesNoReadyOrFinalObject(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	base := newFakeObjectStorage()
	objects := &cancellationFileStorage{fakeObjectStorage: base, cancel: cancel}
	store := newFakeStore()
	svc := NewService(store,
		WithTTS(NewMockTTS()),
		WithObjectStorage(objects),
		WithAudioProcessor(NewMockAudioProcessor()),
	)
	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "First sentence. Second sentence.", "u1")
	_, err := svc.SynthesizeNarration(ctx, nar.ID)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if len(store.assets["c1"]) != 0 {
		t.Fatal("cancellation must not create READY metadata")
	}
	for key := range base.objects {
		if strings.Contains(key, "/audio/") {
			t.Fatalf("cancellation must not upload final object %q", key)
		}
	}
}

func TestNarrationTempDirectoryIsRemovedAfterSuccess(t *testing.T) {
	objects := newFakeObjectStorage()
	store := newFakeStore()
	processor := &pathRecordingProcessor{MockAudioProcessor: NewMockAudioProcessor()}
	svc := NewService(store,
		WithTTS(NewMockTTS()),
		WithObjectStorage(objects),
		WithAudioProcessor(processor),
	)
	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")
	if _, err := svc.SynthesizeNarration(context.Background(), nar.ID); err != nil {
		t.Fatalf("synthesize narration: %v", err)
	}
	if processor.outputPath == "" {
		t.Fatal("expected processor output path")
	}
	if _, err := os.Stat(filepath.Dir(processor.outputPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected staging directory cleanup, stat err=%v", err)
	}
}

type pathRecordingProcessor struct {
	*MockAudioProcessor
	outputPath string
}

func (p *pathRecordingProcessor) ProcessFiles(ctx context.Context, inputPaths []string, outputPath string) (int, error) {
	p.outputPath = outputPath
	return p.MockAudioProcessor.ProcessFiles(ctx, inputPaths, outputPath)
}
