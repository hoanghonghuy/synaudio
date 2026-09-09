package planning

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/audio"
)

type fakeApprovedContentAuthority struct{}

func (f *fakeApprovedContentAuthority) RequireApprovedContentRevision(_ context.Context, _, _ string) error {
	return nil
}

func TestPublishChapterRejectsWhenNewerNarrationAppearsInsidePublishBoundary(t *testing.T) {
	chapterStore := newPublishFakeStore()
	audioStoreBacking := newPublishAudioFakeStore()
	audioSvc := audio.NewService(audioStoreBacking, audio.WithApprovedContentAuthority(&fakeApprovedContentAuthority{}))
	svc := NewService(chapterStore, WithPublishChecker(NewCompositePublishChecker(
		chapterStore,
		&fakePublishContentAuthority{hasApproved: true},
		audioSvc,
		&fakePublishStoryAuthority{},
	)))

	ch, _ := svc.CreateChapter(context.Background(), "s1", "Chapter 1", "u1")
	_, _ = chapterStore.UpdateChapterStatus(context.Background(), ch.ID, "READY")

	audioStoreBacking.narrations[ch.ID] = []audio.NarrationRevision{
		{ID: "nar-1", ChapterID: ch.ID, RevisionNo: 1, SourceContentRevisionID: "cr-1"},
	}
	audioStoreBacking.assets[ch.ID] = []audio.AudioAsset{
		{ID: "asset-1", ChapterID: ch.ID, VersionNo: 1, SourceNarrationRevisionID: "nar-1", Status: "READY", IsActive: true},
	}

	chapterStore.onDuringPublishValidate = func() {
		audioStoreBacking.narrations[ch.ID] = append(audioStoreBacking.narrations[ch.ID], audio.NarrationRevision{
			ID: "nar-2", ChapterID: ch.ID, RevisionNo: 2, SourceContentRevisionID: "cr-1",
		})
	}

	if _, err := svc.PublishChapter(context.Background(), ch.ID); !errors.Is(err, ErrPublishNotReady) {
		t.Fatalf("expected ErrPublishNotReady when newer narration appears inside publish boundary, got %v", err)
	}
	if chapterStore.statuses[ch.ID] == "PUBLISHED" {
		t.Fatal("publish must not commit when active audio is stale for latest narration")
	}
}

type publishAudioFakeStore struct {
	narrations map[string][]audio.NarrationRevision
	assets     map[string][]audio.AudioAsset
}

func newPublishAudioFakeStore() *publishAudioFakeStore {
	return &publishAudioFakeStore{
		narrations: map[string][]audio.NarrationRevision{},
		assets:     map[string][]audio.AudioAsset{},
	}
}

func (s *publishAudioFakeStore) NextNarrationRevision(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (s *publishAudioFakeStore) CreateNarrationRevision(_ context.Context, r audio.NarrationRevision) (audio.NarrationRevision, error) {
	return r, nil
}

func (s *publishAudioFakeStore) GetNarrationRevision(_ context.Context, revisionID string) (audio.NarrationRevision, error) {
	return audio.NarrationRevision{ID: revisionID}, nil
}

func (s *publishAudioFakeStore) GetLatestNarrationRevision(_ context.Context, chapterID string) (audio.NarrationRevision, error) {
	narrations := s.narrations[chapterID]
	if len(narrations) == 0 {
		return audio.NarrationRevision{}, audio.ErrNarrationNotFound
	}
	return narrations[len(narrations)-1], nil
}

func (s *publishAudioFakeStore) CreateTTSSegment(_ context.Context, seg audio.TTSSegment) (audio.TTSSegment, error) {
	return seg, nil
}

func (s *publishAudioFakeStore) GetTTSSegment(_ context.Context, segmentID string) (audio.TTSSegment, error) {
	return audio.TTSSegment{ID: segmentID}, nil
}

func (s *publishAudioFakeStore) UpdateTTSSegment(_ context.Context, seg audio.TTSSegment) (audio.TTSSegment, error) {
	return seg, nil
}

func (s *publishAudioFakeStore) NextAudioVersion(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (s *publishAudioFakeStore) CreateAudioAsset(_ context.Context, a audio.AudioAsset) (audio.AudioAsset, error) {
	return a, nil
}

func (s *publishAudioFakeStore) GetAudioAsset(_ context.Context, assetID string) (audio.AudioAsset, error) {
	for _, assets := range s.assets {
		for _, asset := range assets {
			if asset.ID == assetID {
				return asset, nil
			}
		}
	}
	return audio.AudioAsset{}, audio.ErrAudioAssetNotFound
}

func (s *publishAudioFakeStore) GetActiveAudioAsset(_ context.Context, chapterID string) (audio.AudioAsset, error) {
	for _, asset := range s.assets[chapterID] {
		if asset.IsActive {
			return asset, nil
		}
	}
	return audio.AudioAsset{}, audio.ErrAudioAssetNotFound
}

func (s *publishAudioFakeStore) GetLatestReadyAudioAssetForNarration(_ context.Context, _, _ string) (audio.AudioAsset, error) {
	return audio.AudioAsset{}, audio.ErrReadyAudioAssetNotFound
}

func (s *publishAudioFakeStore) SetActiveAudioAsset(_ context.Context, _, _ string) (audio.AudioAsset, error) {
	return audio.AudioAsset{}, nil
}

func (s *publishAudioFakeStore) SetActiveAudioAssetForLatestNarration(_ context.Context, _, _ string) (audio.AudioAsset, error) {
	return audio.AudioAsset{}, nil
}
