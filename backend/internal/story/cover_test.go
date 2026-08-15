package story_test

import (
	"context"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/story"
)

func TestUploadCoverStoresAssetAndLinksStory(t *testing.T) {
	store := newFakeStore()
	storage := newFakeStorage()
	svc := story.NewService(store, story.WithObjectStorage(storage))

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A"}

	asset, err := svc.UploadCover(context.Background(), story.UploadCoverInput{
		StoryID:     "s1",
		Filename:    "cover.png",
		ContentType: "image/png",
		Data:        []byte("fake-image-bytes"),
		CreatedBy:   "user-1",
	})
	if err != nil {
		t.Fatalf("upload cover: %v", err)
	}

	if asset.ID == "" {
		t.Fatal("expected asset ID")
	}
	if asset.Type != story.AssetTypeCover {
		t.Fatalf("expected type COVER, got %q", asset.Type)
	}
	if asset.StorageKey == "" {
		t.Fatal("expected storage key")
	}
	if asset.Status != story.AssetStatusReady {
		t.Fatalf("expected status READY, got %q", asset.Status)
	}

	// Storage must have received the object.
	if !storage.hasKey(asset.StorageKey) {
		t.Fatalf("expected object stored at key %q", asset.StorageKey)
	}

	// Story must link to the cover asset.
	st := store.stories["s1"]
	if st.CoverAssetID != asset.ID {
		t.Fatalf("expected cover_asset_id %q, got %q", asset.ID, st.CoverAssetID)
	}
}

func TestUploadCoverRejectsUnknownStory(t *testing.T) {
	store := newFakeStore()
	storage := newFakeStorage()
	svc := story.NewService(store, story.WithObjectStorage(storage))

	_, err := svc.UploadCover(context.Background(), story.UploadCoverInput{
		StoryID:     "missing",
		Filename:    "cover.png",
		ContentType: "image/png",
		Data:        []byte("x"),
	})
	if err == nil {
		t.Fatal("expected error for unknown story")
	}
}
