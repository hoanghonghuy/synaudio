package planning

import (
	"context"
	"errors"
	"testing"
)

type fakeStore struct {
	bibles      map[string][]StoryBibleVersion
	nextVersion map[string]int
	endings     map[string][]EndingPlanVersion
	nextEnding  map[string]int
	arcs        map[string][]StoryArc
	arcVersions map[string][]ArcVersion
	nextArcVer  map[string]int
	nextOrdinal map[string]int
	characters  map[string][]Character
	profiles    map[string][]CharacterProfileVersion
	nextProf    map[string]int
	chapters    map[string][]Chapter
	plans       map[string][]ChapterPlanRevision
	nextPlan    map[string]int
	nextChapter map[string]int
	facts       map[string][]StoryFact
	threads     map[string][]PlotThread
	events      map[string][]PlotThreadEvent
	branches    map[string][]CanonBranch
	versions    map[string][]CanonVersion
	nextSeq     map[string]int
	snapshots   map[string][]ContextSnapshot
	changeItems map[string][]CanonChangeItem
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		bibles:      map[string][]StoryBibleVersion{},
		nextVersion: map[string]int{},
		endings:     map[string][]EndingPlanVersion{},
		nextEnding:  map[string]int{},
		arcs:        map[string][]StoryArc{},
		arcVersions: map[string][]ArcVersion{},
		nextArcVer:  map[string]int{},
		nextOrdinal: map[string]int{},
		characters:  map[string][]Character{},
		profiles:    map[string][]CharacterProfileVersion{},
		nextProf:    map[string]int{},
		chapters:    map[string][]Chapter{},
		plans:       map[string][]ChapterPlanRevision{},
		nextPlan:    map[string]int{},
		nextChapter: map[string]int{},
		facts:       map[string][]StoryFact{},
		threads:     map[string][]PlotThread{},
		events:      map[string][]PlotThreadEvent{},
		branches:    map[string][]CanonBranch{},
		versions:    map[string][]CanonVersion{},
		nextSeq:     map[string]int{},
		snapshots:   map[string][]ContextSnapshot{},
		changeItems: map[string][]CanonChangeItem{},
	}
}

func (s *fakeStore) NextBibleVersion(ctx context.Context, storyID string) (int, error) {
	s.nextVersion[storyID]++
	return s.nextVersion[storyID], nil
}

func (s *fakeStore) CreateBibleVersion(ctx context.Context, v StoryBibleVersion) (StoryBibleVersion, error) {
	s.bibles[v.StoryID] = append(s.bibles[v.StoryID], v)
	return v, nil
}

func (s *fakeStore) GetCurrentBible(ctx context.Context, storyID string) (StoryBibleVersion, error) {
	vs := s.bibles[storyID]
	if len(vs) == 0 {
		return StoryBibleVersion{}, ErrBibleNotFound
	}
	return vs[len(vs)-1], nil
}

func (s *fakeStore) NextEndingVersion(ctx context.Context, storyID string) (int, error) {
	s.nextEnding[storyID]++
	return s.nextEnding[storyID], nil
}

func (s *fakeStore) CreateEndingVersion(ctx context.Context, v EndingPlanVersion) (EndingPlanVersion, error) {
	s.endings[v.StoryID] = append(s.endings[v.StoryID], v)
	return v, nil
}

func (s *fakeStore) GetCurrentEnding(ctx context.Context, storyID string) (EndingPlanVersion, error) {
	vs := s.endings[storyID]
	if len(vs) == 0 {
		return EndingPlanVersion{}, ErrEndingNotFound
	}
	return vs[len(vs)-1], nil
}

func (s *fakeStore) NextArcOrdinal(ctx context.Context, storyID string) (int, error) {
	s.nextOrdinal[storyID]++
	return s.nextOrdinal[storyID], nil
}

func (s *fakeStore) CreateArc(ctx context.Context, a StoryArc) (StoryArc, error) {
	s.arcs[a.StoryID] = append(s.arcs[a.StoryID], a)
	return a, nil
}

func (s *fakeStore) NextArcVersion(ctx context.Context, arcID string) (int, error) {
	s.nextArcVer[arcID]++
	return s.nextArcVer[arcID], nil
}

func (s *fakeStore) CreateArcVersion(ctx context.Context, v ArcVersion) (ArcVersion, error) {
	s.arcVersions[v.ArcID] = append(s.arcVersions[v.ArcID], v)
	return v, nil
}

func (s *fakeStore) GetArc(ctx context.Context, arcID string) (StoryArc, error) {
	for _, as := range s.arcs {
		for _, a := range as {
			if a.ID == arcID {
				return a, nil
			}
		}
	}
	return StoryArc{}, ErrArcNotFound
}

func (s *fakeStore) ListArcs(ctx context.Context, storyID string) ([]StoryArc, error) {
	return s.arcs[storyID], nil
}

func (s *fakeStore) CreateCharacter(ctx context.Context, c Character) (Character, error) {
	s.characters[c.StoryID] = append(s.characters[c.StoryID], c)
	return c, nil
}

func (s *fakeStore) NextProfileVersion(ctx context.Context, characterID string) (int, error) {
	s.nextProf[characterID]++
	return s.nextProf[characterID], nil
}

func (s *fakeStore) CreateProfileVersion(ctx context.Context, v CharacterProfileVersion) (CharacterProfileVersion, error) {
	s.profiles[v.CharacterID] = append(s.profiles[v.CharacterID], v)
	return v, nil
}

func (s *fakeStore) ListCharacters(ctx context.Context, storyID string) ([]Character, error) {
	return s.characters[storyID], nil
}

func (s *fakeStore) GetCharacter(ctx context.Context, characterID string) (Character, error) {
	for _, cs := range s.characters {
		for _, c := range cs {
			if c.ID == characterID {
				return c, nil
			}
		}
	}
	return Character{}, ErrCharacterNotFound
}

func (s *fakeStore) NextChapterNumber(ctx context.Context, storyID string) (int, error) {
	s.nextChapter[storyID]++
	return s.nextChapter[storyID], nil
}

func (s *fakeStore) CreateChapter(ctx context.Context, c Chapter) (Chapter, error) {
	s.chapters[c.StoryID] = append(s.chapters[c.StoryID], c)
	return c, nil
}

func (s *fakeStore) NextPlanRevision(ctx context.Context, chapterID string) (int, error) {
	s.nextPlan[chapterID]++
	return s.nextPlan[chapterID], nil
}

func (s *fakeStore) CreatePlanRevision(ctx context.Context, p ChapterPlanRevision) (ChapterPlanRevision, error) {
	s.plans[p.ChapterID] = append(s.plans[p.ChapterID], p)
	return p, nil
}

func (s *fakeStore) GetChapter(ctx context.Context, chapterID string) (Chapter, error) {
	for _, cs := range s.chapters {
		for _, c := range cs {
			if c.ID == chapterID {
				return c, nil
			}
		}
	}
	return Chapter{}, ErrChapterNotFound
}

func (s *fakeStore) ListChapters(ctx context.Context, storyID string) ([]Chapter, error) {
	return s.chapters[storyID], nil
}

func (s *fakeStore) CreateFact(ctx context.Context, f StoryFact) (StoryFact, error) {
	s.facts[f.StoryID] = append(s.facts[f.StoryID], f)
	return f, nil
}

func (s *fakeStore) ListFacts(ctx context.Context, storyID string) ([]StoryFact, error) {
	return s.facts[storyID], nil
}

func (s *fakeStore) CreatePlotThread(ctx context.Context, t PlotThread) (PlotThread, error) {
	s.threads[t.StoryID] = append(s.threads[t.StoryID], t)
	return t, nil
}

func (s *fakeStore) ListPlotThreads(ctx context.Context, storyID string) ([]PlotThread, error) {
	return s.threads[storyID], nil
}

func (s *fakeStore) CreatePlotThreadEvent(ctx context.Context, e PlotThreadEvent) (PlotThreadEvent, error) {
	s.events[e.PlotThreadID] = append(s.events[e.PlotThreadID], e)
	return e, nil
}

func (s *fakeStore) CreateCanonBranch(ctx context.Context, b CanonBranch) (CanonBranch, error) {
	s.branches[b.StoryID] = append(s.branches[b.StoryID], b)
	return b, nil
}

func (s *fakeStore) NextCanonSequence(ctx context.Context, branchID string) (int, error) {
	s.nextSeq[branchID]++
	return s.nextSeq[branchID], nil
}

func (s *fakeStore) CreateCanonVersion(ctx context.Context, v CanonVersion) (CanonVersion, error) {
	s.versions[v.BranchID] = append(s.versions[v.BranchID], v)
	return v, nil
}

func (s *fakeStore) ListCanonVersions(ctx context.Context, branchID string) ([]CanonVersion, error) {
	return s.versions[branchID], nil
}

func (s *fakeStore) CreateContextSnapshot(ctx context.Context, sn ContextSnapshot) (ContextSnapshot, error) {
	s.snapshots[sn.StoryID] = append(s.snapshots[sn.StoryID], sn)
	return sn, nil
}

func (s *fakeStore) ListContextSnapshots(ctx context.Context, storyID string) ([]ContextSnapshot, error) {
	return s.snapshots[storyID], nil
}

func (s *fakeStore) CreateCanonChangeItem(ctx context.Context, c CanonChangeItem) (CanonChangeItem, error) {
	s.changeItems[c.CanonVersionID] = append(s.changeItems[c.CanonVersionID], c)
	return c, nil
}

func (s *fakeStore) ListCanonChangeItems(ctx context.Context, canonVersionID string) ([]CanonChangeItem, error) {
	return s.changeItems[canonVersionID], nil
}

func TestCreateBibleVersionAssignsSequentialVersion(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	v1, err := svc.CreateBibleVersion(context.Background(), "s1", map[string]any{"premise": "A"}, "u1")
	if err != nil {
		t.Fatalf("create v1: %v", err)
	}
	if v1.VersionNo != 1 {
		t.Fatalf("expected version 1, got %d", v1.VersionNo)
	}

	v2, err := svc.CreateBibleVersion(context.Background(), "s1", map[string]any{"premise": "B"}, "u1")
	if err != nil {
		t.Fatalf("create v2: %v", err)
	}
	if v2.VersionNo != 2 {
		t.Fatalf("expected version 2, got %d", v2.VersionNo)
	}
}

func TestCreateBibleVersionRejectsEmptyContent(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.CreateBibleVersion(context.Background(), "s1", nil, "u1"); err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestGetCurrentBibleReturnsLatest(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	_, _ = svc.CreateBibleVersion(context.Background(), "s1", map[string]any{"premise": "A"}, "u1")
	_, _ = svc.CreateBibleVersion(context.Background(), "s1", map[string]any{"premise": "B"}, "u1")

	cur, err := svc.GetCurrentBible(context.Background(), "s1")
	if err != nil {
		t.Fatalf("get current: %v", err)
	}
	if cur.VersionNo != 2 {
		t.Fatalf("expected version 2, got %d", cur.VersionNo)
	}
}

func TestGetCurrentBibleReturnsNotFound(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.GetCurrentBible(context.Background(), "s1"); !errors.Is(err, ErrBibleNotFound) {
		t.Fatalf("expected ErrBibleNotFound, got %v", err)
	}
}
