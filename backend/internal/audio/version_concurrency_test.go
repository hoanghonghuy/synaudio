package audio

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// fakeObjectStorage implements the optional compensating cleanup capability in
// tests without widening the core ObjectStorage contract.
func (s *fakeObjectStorage) Delete(_ context.Context, key string) error {
	delete(s.objects, key)
	return nil
}

type concurrentReservationStore struct {
	*fakeStore
	mu sync.Mutex
}

func (s *concurrentReservationStore) CreateNarrationRevisionAtomically(ctx context.Context, r NarrationRevision) (NarrationRevision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	revisionNo, err := s.fakeStore.NextNarrationRevision(ctx, r.ChapterID)
	if err != nil {
		return NarrationRevision{}, err
	}
	r.RevisionNo = revisionNo
	return s.fakeStore.CreateNarrationRevision(ctx, r)
}

func (s *concurrentReservationStore) CreateAudioAssetAtomically(ctx context.Context, a AudioAsset) (AudioAsset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	versionNo, err := s.fakeStore.NextAudioVersion(ctx, a.ChapterID)
	if err != nil {
		return AudioAsset{}, err
	}
	a.VersionNo = versionNo
	return s.fakeStore.CreateAudioAsset(ctx, a)
}

func TestConcurrentNarrationRevisionAllocationProducesDistinctVersions(t *testing.T) {
	store := &concurrentReservationStore{fakeStore: newFakeStore()}
	svc := NewService(store)
	const attempts = 32

	versions := make(chan int, attempts)
	errs := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := svc.CreateNarrationRevision(context.Background(), "chapter-1", "content-1", "voice-1", "script", "user-1")
			if err != nil {
				errs <- err
				return
			}
			versions <- r.RevisionNo
		}()
	}
	wg.Wait()
	close(errs)
	close(versions)

	for err := range errs {
		t.Fatalf("concurrent narration create failed: %v", err)
	}
	seen := map[int]bool{}
	for version := range versions {
		if seen[version] {
			t.Fatalf("duplicate narration revision %d", version)
		}
		seen[version] = true
	}
	if len(seen) != attempts {
		t.Fatalf("expected %d distinct revisions, got %d", attempts, len(seen))
	}
}

func TestConcurrentAudioVersionAllocationProducesDistinctVersions(t *testing.T) {
	store := &concurrentReservationStore{fakeStore: newFakeStore()}
	svc := NewService(store)
	const attempts = 32

	versions := make(chan int, attempts)
	errs := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a, err := svc.CreateAudioAsset(context.Background(), "chapter-1", "narration-1", "ignored", "audio/mpeg", 1, 1, 96)
			if err != nil {
				errs <- err
				return
			}
			versions <- a.VersionNo
		}()
	}
	wg.Wait()
	close(errs)
	close(versions)

	for err := range errs {
		t.Fatalf("concurrent audio create failed: %v", err)
	}
	seen := map[int]bool{}
	for version := range versions {
		if seen[version] {
			t.Fatalf("duplicate audio version %d", version)
		}
		seen[version] = true
	}
	if len(seen) != attempts {
		t.Fatalf("expected %d distinct audio versions, got %d", attempts, len(seen))
	}
}

type metadataFailStore struct {
	*fakeStore
}

func (s *metadataFailStore) CreateAudioAsset(context.Context, AudioAsset) (AudioAsset, error) {
	return AudioAsset{}, errors.New("metadata unavailable")
}

func TestSynthesizeNarrationCleansOwnObjectWhenReadyMetadataFails(t *testing.T) {
	objects := newFakeObjectStorage()
	store := &metadataFailStore{fakeStore: newFakeStore()}
	processor := &recordingProcessor{output: []byte("FINAL-AUDIO")}
	svc := NewService(store,
		WithTTS(NewMockTTS()),
		WithObjectStorage(objects),
		WithAudioProcessor(processor),
	)

	nar, err := svc.CreateNarrationRevision(context.Background(), "chapter-1", "content-1", "voice-1", "Hello.", "user-1")
	if err != nil {
		t.Fatalf("create narration: %v", err)
	}
	if _, err := svc.SynthesizeNarration(context.Background(), nar.ID); err == nil {
		t.Fatal("expected metadata registration failure")
	}
	for key := range objects.objects {
		if len(key) >= len("chapters/") && key[:len("chapters/")] == "chapters/" {
			t.Fatalf("failed attempt left final audio object %q", key)
		}
	}
}
