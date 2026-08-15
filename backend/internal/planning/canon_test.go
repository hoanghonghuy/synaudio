package planning

import (
	"context"
	"testing"
)

type canonFakeStore struct {
	*fakeStore
	branches map[string][]CanonBranch
	versions map[string][]CanonVersion
	nextSeq  map[string]int
}

func newCanonFakeStore() *canonFakeStore {
	return &canonFakeStore{
		fakeStore: newFakeStore(),
		branches:  map[string][]CanonBranch{},
		versions:  map[string][]CanonVersion{},
		nextSeq:   map[string]int{},
	}
}

func (s *canonFakeStore) CreateCanonBranch(ctx context.Context, b CanonBranch) (CanonBranch, error) {
	s.branches[b.StoryID] = append(s.branches[b.StoryID], b)
	return b, nil
}

func (s *canonFakeStore) NextCanonSequence(ctx context.Context, branchID string) (int, error) {
	s.nextSeq[branchID]++
	return s.nextSeq[branchID], nil
}

func (s *canonFakeStore) CreateCanonVersion(ctx context.Context, v CanonVersion) (CanonVersion, error) {
	s.versions[v.BranchID] = append(s.versions[v.BranchID], v)
	return v, nil
}

func (s *canonFakeStore) ListCanonVersions(ctx context.Context, branchID string) ([]CanonVersion, error) {
	return s.versions[branchID], nil
}

func TestCreateCanonBranch(t *testing.T) {
	store := newCanonFakeStore()
	svc := NewService(store)

	b, err := svc.CreateCanonBranch(context.Background(), "s1", "OFFICIAL")
	if err != nil {
		t.Fatalf("create branch: %v", err)
	}
	if b.Type != "OFFICIAL" {
		t.Fatalf("expected OFFICIAL, got %q", b.Type)
	}
	if b.Status != "ACTIVE" {
		t.Fatalf("expected ACTIVE, got %q", b.Status)
	}
}

func TestCreateCanonVersionAssignsSequence(t *testing.T) {
	store := newCanonFakeStore()
	svc := NewService(store)

	b, _ := svc.CreateCanonBranch(context.Background(), "s1", "OFFICIAL")

	v1, err := svc.CreateCanonVersion(context.Background(), "s1", b.ID, "c1", "u1")
	if err != nil {
		t.Fatalf("create version 1: %v", err)
	}
	if v1.SequenceNo != 1 {
		t.Fatalf("expected sequence 1, got %d", v1.SequenceNo)
	}

	v2, err := svc.CreateCanonVersion(context.Background(), "s1", b.ID, "c2", "u1")
	if err != nil {
		t.Fatalf("create version 2: %v", err)
	}
	if v2.SequenceNo != 2 {
		t.Fatalf("expected sequence 2, got %d", v2.SequenceNo)
	}
}

func TestListCanonVersionsReturnsAll(t *testing.T) {
	store := newCanonFakeStore()
	svc := NewService(store)

	b, _ := svc.CreateCanonBranch(context.Background(), "s1", "OFFICIAL")
	_, _ = svc.CreateCanonVersion(context.Background(), "s1", b.ID, "c1", "u1")
	_, _ = svc.CreateCanonVersion(context.Background(), "s1", b.ID, "c2", "u1")

	versions, err := svc.ListCanonVersions(context.Background(), b.ID)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
}
