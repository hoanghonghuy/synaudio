package audio

import (
	"context"
	"errors"
	"testing"
)

func TestGetListenerAudioURLRejectsWhenNewerNarrationAppearsBetweenValidationAndPresign(t *testing.T) {
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
