package planning

import (
	"context"
	"errors"
	"testing"
)

type endingFakeStore struct {
	*fakeStore
	endings     map[string][]EndingPlanVersion
	nextEnding  map[string]int
}

func newEndingFakeStore() *endingFakeStore {
	return &endingFakeStore{
		fakeStore:  newFakeStore(),
		endings:    map[string][]EndingPlanVersion{},
		nextEnding: map[string]int{},
	}
}

func (s *endingFakeStore) NextEndingVersion(ctx context.Context, storyID string) (int, error) {
	s.nextEnding[storyID]++
	return s.nextEnding[storyID], nil
}

func (s *endingFakeStore) CreateEndingVersion(ctx context.Context, v EndingPlanVersion) (EndingPlanVersion, error) {
	s.endings[v.StoryID] = append(s.endings[v.StoryID], v)
	return v, nil
}

func (s *endingFakeStore) GetCurrentEnding(ctx context.Context, storyID string) (EndingPlanVersion, error) {
	vs := s.endings[storyID]
	if len(vs) == 0 {
		return EndingPlanVersion{}, ErrEndingNotFound
	}
	return vs[len(vs)-1], nil
}

func TestCreateEndingVersionAssignsSequentialVersion(t *testing.T) {
	store := newEndingFakeStore()
	svc := NewService(store)

	v1, err := svc.CreateEndingVersion(context.Background(), "s1", map[string]any{"ending": "A"}, "u1")
	if err != nil {
		t.Fatalf("create v1: %v", err)
	}
	if v1.VersionNo != 1 {
		t.Fatalf("expected version 1, got %d", v1.VersionNo)
	}

	v2, err := svc.CreateEndingVersion(context.Background(), "s1", map[string]any{"ending": "B"}, "u1")
	if err != nil {
		t.Fatalf("create v2: %v", err)
	}
	if v2.VersionNo != 2 {
		t.Fatalf("expected version 2, got %d", v2.VersionNo)
	}
}

func TestCreateEndingVersionRejectsEmptyContent(t *testing.T) {
	store := newEndingFakeStore()
	svc := NewService(store)

	if _, err := svc.CreateEndingVersion(context.Background(), "s1", nil, "u1"); err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestGetCurrentEndingReturnsLatest(t *testing.T) {
	store := newEndingFakeStore()
	svc := NewService(store)

	_, _ = svc.CreateEndingVersion(context.Background(), "s1", map[string]any{"ending": "A"}, "u1")
	_, _ = svc.CreateEndingVersion(context.Background(), "s1", map[string]any{"ending": "B"}, "u1")

	cur, err := svc.GetCurrentEnding(context.Background(), "s1")
	if err != nil {
		t.Fatalf("get current: %v", err)
	}
	if cur.VersionNo != 2 {
		t.Fatalf("expected version 2, got %d", cur.VersionNo)
	}
}

func TestGetCurrentEndingReturnsNotFound(t *testing.T) {
	store := newEndingFakeStore()
	svc := NewService(store)

	if _, err := svc.GetCurrentEnding(context.Background(), "s1"); !errors.Is(err, ErrEndingNotFound) {
		t.Fatalf("expected ErrEndingNotFound, got %v", err)
	}
}
