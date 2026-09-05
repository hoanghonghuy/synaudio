package audio

import (
	"context"
	"errors"
	"testing"
)

func mustCreateAudioAsset(t *testing.T, svc *Service, chapterID string) AudioAsset {
	t.Helper()
	asset, err := svc.CreateAudioAsset(context.Background(), chapterID, "nar-1", "key", "audio/mpeg", 1, 1000, 96)
	if err != nil {
		t.Fatalf("create audio asset: %v", err)
	}
	return asset
}

func mustActiveAsset(t *testing.T, svc *Service, chapterID string) AudioAsset {
	t.Helper()
	asset, err := svc.GetActiveAudioAsset(context.Background(), chapterID)
	if err != nil {
		t.Fatalf("get active audio asset: %v", err)
	}
	return asset
}

func TestActivateAudioAssetCrossChapterIsNoOp(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	current := mustCreateAudioAsset(t, svc, "chapter-a")
	if _, err := svc.ActivateAudioAsset(context.Background(), "chapter-a", current.ID); err != nil {
		t.Fatalf("activate current asset: %v", err)
	}
	foreign := mustCreateAudioAsset(t, svc, "chapter-b")

	if _, err := svc.ActivateAudioAsset(context.Background(), "chapter-a", foreign.ID); !errors.Is(err, ErrAudioAssetNotFound) {
		t.Fatalf("expected not found for cross-chapter asset, got %v", err)
	}
	if got := mustActiveAsset(t, svc, "chapter-a"); got.ID != current.ID {
		t.Fatalf("cross-chapter failure changed active asset: got %q want %q", got.ID, current.ID)
	}
}

func TestActivateAudioAssetUnknownIsNoOp(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	current := mustCreateAudioAsset(t, svc, "chapter-a")
	if _, err := svc.ActivateAudioAsset(context.Background(), "chapter-a", current.ID); err != nil {
		t.Fatalf("activate current asset: %v", err)
	}

	if _, err := svc.ActivateAudioAsset(context.Background(), "chapter-a", "missing"); !errors.Is(err, ErrAudioAssetNotFound) {
		t.Fatalf("expected not found for unknown asset, got %v", err)
	}
	if got := mustActiveAsset(t, svc, "chapter-a"); got.ID != current.ID {
		t.Fatalf("unknown-asset failure changed active asset: got %q want %q", got.ID, current.ID)
	}
}

func TestActivateAudioAssetRejectsNonReadyWithoutChangingCurrent(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	current := mustCreateAudioAsset(t, svc, "chapter-a")
	if _, err := svc.ActivateAudioAsset(context.Background(), "chapter-a", current.ID); err != nil {
		t.Fatalf("activate current asset: %v", err)
	}

	candidate := mustCreateAudioAsset(t, svc, "chapter-a")
	for i := range store.assets["chapter-a"] {
		if store.assets["chapter-a"][i].ID == candidate.ID {
			store.assets["chapter-a"][i].Status = "FAILED"
		}
	}

	if _, err := svc.ActivateAudioAsset(context.Background(), "chapter-a", candidate.ID); !errors.Is(err, ErrAudioAssetNotFound) {
		t.Fatalf("expected not found/domain rejection for non-ready asset, got %v", err)
	}
	if got := mustActiveAsset(t, svc, "chapter-a"); got.ID != current.ID {
		t.Fatalf("non-ready failure changed active asset: got %q want %q", got.ID, current.ID)
	}
}

func TestActivateAudioAssetReplacesCurrentReadyAsset(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	first := mustCreateAudioAsset(t, svc, "chapter-a")
	second := mustCreateAudioAsset(t, svc, "chapter-a")
	if _, err := svc.ActivateAudioAsset(context.Background(), "chapter-a", first.ID); err != nil {
		t.Fatalf("activate first asset: %v", err)
	}

	activated, err := svc.ActivateAudioAsset(context.Background(), "chapter-a", second.ID)
	if err != nil {
		t.Fatalf("activate replacement: %v", err)
	}
	if activated.ID != second.ID || !activated.IsActive {
		t.Fatalf("unexpected activated asset: %+v", activated)
	}
	if got := mustActiveAsset(t, svc, "chapter-a"); got.ID != second.ID {
		t.Fatalf("expected replacement %q active, got %q", second.ID, got.ID)
	}
	old, err := store.GetAudioAsset(context.Background(), first.ID)
	if err != nil {
		t.Fatalf("get prior asset: %v", err)
	}
	if old.IsActive {
		t.Fatal("expected prior asset to be inactive after replacement")
	}
}
