package planning

import (
	"context"
	"errors"
	"testing"
)

type arcFakeStore struct {
	*fakeStore
	arcs        map[string][]StoryArc
	arcVersions map[string][]ArcVersion
	nextArcVer  map[string]int
}

func newArcFakeStore() *arcFakeStore {
	return &arcFakeStore{
		fakeStore:   newFakeStore(),
		arcs:        map[string][]StoryArc{},
		arcVersions: map[string][]ArcVersion{},
		nextArcVer:  map[string]int{},
	}
}

func (s *arcFakeStore) CreateArc(ctx context.Context, a StoryArc) (StoryArc, error) {
	s.arcs[a.StoryID] = append(s.arcs[a.StoryID], a)
	return a, nil
}

func (s *arcFakeStore) NextArcVersion(ctx context.Context, arcID string) (int, error) {
	s.nextArcVer[arcID]++
	return s.nextArcVer[arcID], nil
}

func (s *arcFakeStore) CreateArcVersion(ctx context.Context, v ArcVersion) (ArcVersion, error) {
	s.arcVersions[v.ArcID] = append(s.arcVersions[v.ArcID], v)
	return v, nil
}

func (s *arcFakeStore) GetArc(ctx context.Context, arcID string) (StoryArc, error) {
	for _, as := range s.arcs {
		for _, a := range as {
			if a.ID == arcID {
				return a, nil
			}
		}
	}
	return StoryArc{}, ErrArcNotFound
}

func (s *arcFakeStore) ListArcs(ctx context.Context, storyID string) ([]StoryArc, error) {
	return s.arcs[storyID], nil
}

func TestCreateArcAssignsOrdinal(t *testing.T) {
	store := newArcFakeStore()
	svc := NewService(store)

	a1, err := svc.CreateArc(context.Background(), "s1", map[string]any{"objective": "A"}, "u1")
	if err != nil {
		t.Fatalf("create arc 1: %v", err)
	}
	if a1.Ordinal != 1 {
		t.Fatalf("expected ordinal 1, got %d", a1.Ordinal)
	}

	a2, err := svc.CreateArc(context.Background(), "s1", map[string]any{"objective": "B"}, "u1")
	if err != nil {
		t.Fatalf("create arc 2: %v", err)
	}
	if a2.Ordinal != 2 {
		t.Fatalf("expected ordinal 2, got %d", a2.Ordinal)
	}
}

func TestCreateArcRejectsEmptyContent(t *testing.T) {
	store := newArcFakeStore()
	svc := NewService(store)

	if _, err := svc.CreateArc(context.Background(), "s1", nil, "u1"); err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestListArcsReturnsOrdered(t *testing.T) {
	store := newArcFakeStore()
	svc := NewService(store)

	_, _ = svc.CreateArc(context.Background(), "s1", map[string]any{"objective": "A"}, "u1")
	_, _ = svc.CreateArc(context.Background(), "s1", map[string]any{"objective": "B"}, "u1")

	arcs, err := svc.ListArcs(context.Background(), "s1")
	if err != nil {
		t.Fatalf("list arcs: %v", err)
	}
	if len(arcs) != 2 {
		t.Fatalf("expected 2 arcs, got %d", len(arcs))
	}
}

func TestGetArcReturnsNotFound(t *testing.T) {
	store := newArcFakeStore()
	svc := NewService(store)

	if _, err := svc.GetArc(context.Background(), "missing"); !errors.Is(err, ErrArcNotFound) {
		t.Fatalf("expected ErrArcNotFound, got %v", err)
	}
}
