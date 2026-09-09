package audio

import (
	"context"
	"errors"
	"sync"
)

type fakeStore struct {
	narrations map[string][]NarrationRevision
	nextNar    map[string]int
	segments   map[string][]TTSSegment
	assets     map[string][]AudioAsset
	nextVer    map[string]int

	listenerMu                      sync.Mutex
	onBeforeListenerAudioValidation func(*fakeStore)
	onBeforeListenerPresign         func(*fakeStore, AudioAsset)

	chapterStatuses map[string]string
	storyByChapter  map[string]string
	storyVisibility map[string]string
	storyStatus     map[string]string
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		narrations: map[string][]NarrationRevision{},
		nextNar:    map[string]int{},
		segments:   map[string][]TTSSegment{},
		assets:     map[string][]AudioAsset{},
		nextVer:    map[string]int{},
	}
}

func (s *fakeStore) NextNarrationRevision(_ context.Context, chapterID string) (int, error) {
	s.nextNar[chapterID]++
	return s.nextNar[chapterID], nil
}

func (s *fakeStore) CreateNarrationRevision(_ context.Context, r NarrationRevision) (NarrationRevision, error) {
	s.narrations[r.ChapterID] = append(s.narrations[r.ChapterID], r)
	return r, nil
}

func (s *fakeStore) CreateNarrationRevisionAtomically(ctx context.Context, r NarrationRevision) (NarrationRevision, error) {
	s.listenerMu.Lock()
	defer s.listenerMu.Unlock()

	revisionNo, err := s.NextNarrationRevision(ctx, r.ChapterID)
	if err != nil {
		return NarrationRevision{}, err
	}
	r.RevisionNo = revisionNo
	return s.CreateNarrationRevision(ctx, r)
}

func (s *fakeStore) GetNarrationRevision(_ context.Context, revisionID string) (NarrationRevision, error) {
	for _, rs := range s.narrations {
		for _, r := range rs {
			if r.ID == revisionID {
				return r, nil
			}
		}
	}
	return NarrationRevision{}, ErrNarrationNotFound
}

func (s *fakeStore) GetLatestNarrationRevision(_ context.Context, chapterID string) (NarrationRevision, error) {
	rs := s.narrations[chapterID]
	if len(rs) == 0 {
		return NarrationRevision{}, ErrNarrationNotFound
	}
	latest := rs[0]
	for _, r := range rs[1:] {
		if r.RevisionNo > latest.RevisionNo {
			latest = r
		}
	}
	return latest, nil
}

func (s *fakeStore) CreateTTSSegment(_ context.Context, seg TTSSegment) (TTSSegment, error) {
	s.segments[seg.NarrationRevisionID] = append(s.segments[seg.NarrationRevisionID], seg)
	return seg, nil
}

func (s *fakeStore) GetTTSSegment(_ context.Context, segmentID string) (TTSSegment, error) {
	for _, segs := range s.segments {
		for _, seg := range segs {
			if seg.ID == segmentID {
				return seg, nil
			}
		}
	}
	return TTSSegment{}, ErrTTSSegmentNotFound
}

func (s *fakeStore) UpdateTTSSegment(_ context.Context, seg TTSSegment) (TTSSegment, error) {
	for narID, segs := range s.segments {
		for i, s2 := range segs {
			if s2.ID == seg.ID {
				s.segments[narID][i] = seg
				return seg, nil
			}
		}
	}
	return TTSSegment{}, ErrTTSSegmentNotFound
}

func (s *fakeStore) NextAudioVersion(_ context.Context, chapterID string) (int, error) {
	s.nextVer[chapterID]++
	return s.nextVer[chapterID], nil
}

func (s *fakeStore) CreateAudioAsset(_ context.Context, a AudioAsset) (AudioAsset, error) {
	s.assets[a.ChapterID] = append(s.assets[a.ChapterID], a)
	return a, nil
}

func (s *fakeStore) GetAudioAsset(_ context.Context, assetID string) (AudioAsset, error) {
	for _, as := range s.assets {
		for _, a := range as {
			if a.ID == assetID {
				return a, nil
			}
		}
	}
	return AudioAsset{}, ErrAudioAssetNotFound
}

func (s *fakeStore) GetActiveAudioAsset(_ context.Context, chapterID string) (AudioAsset, error) {
	for _, a := range s.assets[chapterID] {
		if a.IsActive {
			return a, nil
		}
	}
	return AudioAsset{}, ErrAudioAssetNotFound
}

func (s *fakeStore) GetLatestReadyAudioAssetForNarration(_ context.Context, chapterID, narrationRevisionID string) (AudioAsset, error) {
	var latest AudioAsset
	found := false
	for _, a := range s.assets[chapterID] {
		if a.Status != "READY" || a.IsActive || a.SourceNarrationRevisionID != narrationRevisionID {
			continue
		}
		if !found || a.VersionNo > latest.VersionNo {
			latest = a
			found = true
		}
	}
	if !found {
		return AudioAsset{}, ErrReadyAudioAssetNotFound
	}
	return latest, nil
}

func (s *fakeStore) SetActiveAudioAssetForLatestNarration(ctx context.Context, chapterID, assetID string) (AudioAsset, error) {
	latest, err := s.GetLatestNarrationRevision(ctx, chapterID)
	if err != nil {
		return AudioAsset{}, err
	}
	asset, err := s.GetAudioAsset(ctx, assetID)
	if err != nil {
		return AudioAsset{}, err
	}
	if asset.ChapterID != chapterID || asset.Status != "READY" {
		return AudioAsset{}, ErrAudioAssetNotFound
	}
	if asset.SourceNarrationRevisionID != latest.ID {
		return AudioAsset{}, ErrAudioAssetStaleForNarration
	}
	return s.SetActiveAudioAsset(ctx, chapterID, assetID)
}

func (s *fakeStore) IssueListenerEligibleAudioURL(
	ctx context.Context,
	chapterID string,
	check ListenerEligibilityChecker,
	issue ListenerEligibleAudioIssuer,
) (string, error) {
	s.listenerMu.Lock()
	defer s.listenerMu.Unlock()

	if check != nil {
		if err := check(ctx, chapterID); err != nil {
			return "", err
		}
	}

	asset, err := s.getListenerEligibleActiveAudioLocked(ctx, chapterID)
	if err != nil {
		return "", err
	}

	if s.onBeforeListenerPresign != nil {
		s.onBeforeListenerPresign(s, asset)
	}

	return issue(ctx, asset)
}

func (s *fakeStore) getListenerEligibleActiveAudioLocked(ctx context.Context, chapterID string) (AudioAsset, error) {
	latestBefore, err := s.GetLatestNarrationRevision(ctx, chapterID)
	if err != nil {
		if errors.Is(err, ErrNarrationNotFound) {
			return AudioAsset{}, ErrListenerAudioNotEligible
		}
		return AudioAsset{}, err
	}

	if s.onBeforeListenerAudioValidation != nil {
		s.onBeforeListenerAudioValidation(s)
	}

	asset, err := s.GetActiveAudioAsset(ctx, chapterID)
	if err != nil {
		if errors.Is(err, ErrAudioAssetNotFound) {
			return AudioAsset{}, ErrListenerAudioNotEligible
		}
		return AudioAsset{}, err
	}
	if asset.Status != "READY" {
		return AudioAsset{}, ErrListenerAudioNotEligible
	}

	latestAfter, err := s.GetLatestNarrationRevision(ctx, chapterID)
	if err != nil {
		if errors.Is(err, ErrNarrationNotFound) {
			return AudioAsset{}, ErrListenerAudioNotEligible
		}
		return AudioAsset{}, err
	}
	if latestAfter.ID != latestBefore.ID || asset.SourceNarrationRevisionID != latestAfter.ID {
		return AudioAsset{}, ErrListenerAudioNotEligible
	}

	return asset, nil
}

func (s *fakeStore) SetActiveAudioAsset(_ context.Context, chapterID, assetID string) (AudioAsset, error) {
	targetIndex := -1
	for i, a := range s.assets[chapterID] {
		if a.ID == assetID && a.Status == "READY" {
			targetIndex = i
			break
		}
	}
	if targetIndex < 0 {
		return AudioAsset{}, ErrAudioAssetNotFound
	}

	var activated AudioAsset
	for i, a := range s.assets[chapterID] {
		a.IsActive = i == targetIndex
		s.assets[chapterID][i] = a
		if a.IsActive {
			activated = a
		}
	}
	return activated, nil
}

func (s *fakeStore) revokeStoryEligibility(storyID string) {
	s.listenerMu.Lock()
	defer s.listenerMu.Unlock()
	if s.storyVisibility == nil {
		s.storyVisibility = map[string]string{}
	}
	s.storyVisibility[storyID] = "PRIVATE"
}
