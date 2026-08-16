package audio

import (
	"context"
)

type fakeStore struct {
	narrations map[string][]NarrationRevision
	nextNar    map[string]int
	segments   map[string][]TTSSegment
	assets     map[string][]AudioAsset
	nextVer    map[string]int
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

func (s *fakeStore) SetActiveAudioAsset(_ context.Context, chapterID, assetID string) (AudioAsset, error) {
	var activated AudioAsset
	for i, a := range s.assets[chapterID] {
		if a.ID == assetID {
			a.IsActive = true
			activated = a
			s.assets[chapterID][i] = a
		} else {
			a.IsActive = false
			s.assets[chapterID][i] = a
		}
	}
	if activated.ID == "" {
		return AudioAsset{}, ErrAudioAssetNotFound
	}
	return activated, nil
}
