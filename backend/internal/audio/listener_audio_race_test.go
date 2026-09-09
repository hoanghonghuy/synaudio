package audio

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetListenerAudioURLRejectsWhenNewerNarrationAppearsBeforeSelection(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(
		store,
		WithPresigner(fakePresigner{url: "https://cdn.example.com"}),
		WithListenerAudioGate(&fakeListenerAudioGate{}),
	)

	store.narrations["c1"] = []NarrationRevision{
		{ID: "nar-1", ChapterID: "c1", RevisionNo: 1, SourceContentRevisionID: "cr-1"},
	}
	store.assets["c1"] = []AudioAsset{
		{
			ID:                        "asset-1",
			ChapterID:                 "c1",
			VersionNo:                 1,
			SourceNarrationRevisionID: "nar-1",
			Status:                    "READY",
			StorageKey:                "audio/c1/v1.mp3",
			IsActive:                  true,
		},
	}
	store.onBeforeListenerAudioValidation = func(fs *fakeStore) {
		fs.narrations["c1"] = append(fs.narrations["c1"], NarrationRevision{
			ID: "nar-2", ChapterID: "c1", RevisionNo: 2, SourceContentRevisionID: "cr-1",
		})
	}

	if _, err := svc.GetListenerAudioURL(context.Background(), "c1"); !errors.Is(err, ErrListenerAudioNotEligible) {
		t.Fatalf("expected ErrListenerAudioNotEligible when newer narration interleaves listener validation, got %v", err)
	}
}

func TestGetListenerAudioURLLinearizesBeforeConcurrentNarrationCanStaleSelection(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(
		store,
		WithPresigner(fakePresigner{url: "https://cdn.example.com"}),
		WithListenerAudioGate(&fakeListenerAudioGate{}),
	)

	store.narrations["c1"] = []NarrationRevision{
		{ID: "nar-1", ChapterID: "c1", RevisionNo: 1, SourceContentRevisionID: "cr-1"},
	}
	store.nextNar["c1"] = 1
	store.assets["c1"] = []AudioAsset{
		{
			ID:                        "asset-1",
			ChapterID:                 "c1",
			VersionNo:                 1,
			SourceNarrationRevisionID: "nar-1",
			Status:                    "READY",
			StorageKey:                "audio/c1/v1.mp3",
			IsActive:                  true,
		},
	}

	var narrationCommitted atomic.Bool
	narrationDone := make(chan struct{})

	store.onBeforeListenerPresign = func(fs *fakeStore, asset AudioAsset) {
		if asset.SourceNarrationRevisionID != "nar-1" {
			t.Errorf("expected selection of nar-1 asset before presign, got %q", asset.SourceNarrationRevisionID)
		}

		go func() {
			_, err := fs.CreateNarrationRevisionAtomically(context.Background(), NarrationRevision{
				ID:                      "nar-2",
				ChapterID:               "c1",
				SourceContentRevisionID: "cr-1",
			})
			if err != nil {
				t.Errorf("create concurrent narration revision: %v", err)
			}
			narrationCommitted.Store(true)
			close(narrationDone)
		}()
	}

	url, err := svc.GetListenerAudioURL(context.Background(), "c1")
	if err != nil {
		t.Fatalf("get listener audio url: %v", err)
	}
	if url != "https://cdn.example.com/audio/c1/v1.mp3" {
		t.Fatalf("unexpected url: %q", url)
	}
	if narrationCommitted.Load() {
		t.Fatal("concurrent narration committed before listener URL was linearized")
	}

	latest, err := store.GetLatestNarrationRevision(context.Background(), "c1")
	if err != nil {
		t.Fatalf("get latest narration before concurrent commit completes: %v", err)
	}
	if latest.ID != "nar-1" {
		t.Fatalf("expected latest narration to remain nar-1 until lock release, got %q", latest.ID)
	}

	select {
	case <-narrationDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for concurrent narration to commit after listener lock release")
	}

	latest, err = store.GetLatestNarrationRevision(context.Background(), "c1")
	if err != nil {
		t.Fatalf("get latest narration after concurrent commit: %v", err)
	}
	if latest.ID != "nar-2" {
		t.Fatalf("expected nar-2 after concurrent narration commit, got %q", latest.ID)
	}
}
