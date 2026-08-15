package planning

import (
	"context"
	"testing"
)

type snapshotFakeStore struct {
	*fakeStore
	snapshots map[string][]ContextSnapshot
}

func newSnapshotFakeStore() *snapshotFakeStore {
	return &snapshotFakeStore{
		fakeStore: newFakeStore(),
		snapshots: map[string][]ContextSnapshot{},
	}
}

func (s *snapshotFakeStore) CreateContextSnapshot(ctx context.Context, sn ContextSnapshot) (ContextSnapshot, error) {
	s.snapshots[sn.StoryID] = append(s.snapshots[sn.StoryID], sn)
	return sn, nil
}

func (s *snapshotFakeStore) ListContextSnapshots(ctx context.Context, storyID string) ([]ContextSnapshot, error) {
	return s.snapshots[storyID], nil
}

func TestCreateContextSnapshot(t *testing.T) {
	store := newSnapshotFakeStore()
	svc := NewService(store)

	sn, err := svc.CreateContextSnapshot(context.Background(), "s1", "c1", "bible-1", "ending-1", "arc-1", "profile-1", "prompt-v1", "workflow-v1", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if sn.Provider != "openai" {
		t.Fatalf("expected openai, got %q", sn.Provider)
	}
	if sn.Model != "gpt-4" {
		t.Fatalf("expected gpt-4, got %q", sn.Model)
	}
}

func TestListContextSnapshotsReturnsAll(t *testing.T) {
	store := newSnapshotFakeStore()
	svc := NewService(store)

	_, _ = svc.CreateContextSnapshot(context.Background(), "s1", "c1", "b1", "e1", "a1", "p1", "pv1", "wv1", "openai", "gpt-4")
	_, _ = svc.CreateContextSnapshot(context.Background(), "s1", "c2", "b1", "e1", "a1", "p1", "pv1", "wv1", "openai", "gpt-4")

	snapshots, err := svc.ListContextSnapshots(context.Background(), "s1")
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snapshots))
	}
}
