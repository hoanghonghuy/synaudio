package audio

import (
	"context"
	"testing"
)

func TestCreateNarrationRevisionAssignsSequentialRevision(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	r1, err := svc.CreateNarrationRevision(context.Background(), "c1", "content-rev-1", "voice-1", "script text", "u1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if r1.RevisionNo != 1 {
		t.Fatalf("expected revision 1, got %d", r1.RevisionNo)
	}

	r2, err := svc.CreateNarrationRevision(context.Background(), "c1", "content-rev-2", "voice-1", "script 2", "u1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if r2.RevisionNo != 2 {
		t.Fatalf("expected revision 2, got %d", r2.RevisionNo)
	}
}

func TestCreateNarrationRevisionRejectsEmptyScript(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "v1", "   ", "u1"); err == nil {
		t.Fatal("expected error for empty script")
	}
}

func TestCreateAudioAssetAssignsSequentialVersion(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	a1, err := svc.CreateAudioAsset(context.Background(), "c1", "nar-1", "stories/s1/chapters/c1/audio/v1/chapter.mp3", "audio/mpeg", 1024, 180000, 64)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if a1.VersionNo != 1 {
		t.Fatalf("expected version 1, got %d", a1.VersionNo)
	}

	a2, err := svc.CreateAudioAsset(context.Background(), "c1", "nar-2", "stories/s1/chapters/c1/audio/v2/chapter.mp3", "audio/mpeg", 1024, 180000, 64)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if a2.VersionNo != 2 {
		t.Fatalf("expected version 2, got %d", a2.VersionNo)
	}
}

func TestActivateAudioAssetPromotesVersion(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	a1, _ := svc.CreateAudioAsset(context.Background(), "c1", "nar-1", "key1", "audio/mpeg", 1024, 180000, 64)
	a2, _ := svc.CreateAudioAsset(context.Background(), "c1", "nar-2", "key2", "audio/mpeg", 1024, 180000, 64)

	activated, err := svc.ActivateAudioAsset(context.Background(), "c1", a2.ID)
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	if !activated.IsActive {
		t.Fatal("expected activated asset to be active")
	}

	// a1 should no longer be active.
	got1, _ := store.GetAudioAsset(context.Background(), a1.ID)
	if got1.IsActive {
		t.Fatal("expected a1 to be inactive after promoting a2")
	}
}

func TestGetActiveAudioAssetReturnsActive(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	a1, _ := svc.CreateAudioAsset(context.Background(), "c1", "nar-1", "key1", "audio/mpeg", 1024, 180000, 64)
	_, _ = svc.ActivateAudioAsset(context.Background(), "c1", a1.ID)

	active, err := svc.GetActiveAudioAsset(context.Background(), "c1")
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if active.ID != a1.ID {
		t.Fatalf("expected %q, got %q", a1.ID, active.ID)
	}
}
