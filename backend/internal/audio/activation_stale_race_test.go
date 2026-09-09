package audio

import (
	"context"
	"errors"
	"testing"
)

type interleavingFakeStore struct {
	*fakeStore
	onBeforeLatestNarrationCheck func(*fakeStore)
}

func (s *interleavingFakeStore) SetActiveAudioAssetForLatestNarration(ctx context.Context, chapterID, assetID string) (AudioAsset, error) {
	if s.onBeforeLatestNarrationCheck != nil {
		s.onBeforeLatestNarrationCheck(s.fakeStore)
	}
	return s.fakeStore.SetActiveAudioAssetForLatestNarration(ctx, chapterID, assetID)
}

func TestActivateAudioAssetForChapterRejectsWhenNewerNarrationAppearsAtActivationBoundary(t *testing.T) {
	base := newFakeStore()
	store := &interleavingFakeStore{fakeStore: base}
	svc := NewService(store)
	store.narrations["c1"] = []NarrationRevision{
		{ID: "nar-1", ChapterID: "c1", RevisionNo: 1},
	}
	store.assets["c1"] = []AudioAsset{
		{ID: "a1", ChapterID: "c1", VersionNo: 1, SourceNarrationRevisionID: "nar-1", Status: "READY", IsActive: false},
	}
	store.onBeforeLatestNarrationCheck = func(fs *fakeStore) {
		fs.narrations["c1"] = append(fs.narrations["c1"], NarrationRevision{
			ID: "nar-2", ChapterID: "c1", RevisionNo: 2,
		})
	}

	if _, err := svc.ActivateAudioAssetForChapter(context.Background(), "c1", "a1"); !errors.Is(err, ErrAudioAssetStaleForNarration) {
		t.Fatalf("expected stale-for-narration rejection at activation boundary, got %v", err)
	}
	for _, asset := range store.assets["c1"] {
		if asset.IsActive {
			t.Fatalf("stale activation must not commit; asset %q became active", asset.ID)
		}
	}
}
