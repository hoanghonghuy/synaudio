package planning

import (
	"context"
	"errors"
	"testing"
)

func activeOfficialFrom(branches []CanonBranch) (CanonBranch, error) {
	var found CanonBranch
	matches := 0
	for _, branch := range branches {
		if branch.Type == "OFFICIAL" && branch.Status == "ACTIVE" {
			found = branch
			matches++
		}
	}
	if matches != 1 {
		return CanonBranch{}, ErrCanonBranchNotFound
	}
	return found, nil
}

func (s *fakeStore) GetActiveOfficialCanonBranch(_ context.Context, storyID string) (CanonBranch, error) {
	return activeOfficialFrom(s.branches[storyID])
}

func (s *canonFakeStore) GetActiveOfficialCanonBranch(_ context.Context, storyID string) (CanonBranch, error) {
	return activeOfficialFrom(s.branches[storyID])
}

func TestGetActiveOfficialCanonBranchSelectsOnlyActiveOfficial(t *testing.T) {
	store := newCanonFakeStore()
	store.branches["s1"] = []CanonBranch{
		{ID: "official-archived", StoryID: "s1", Type: "OFFICIAL", Status: "ARCHIVED"},
		{ID: "retcon-active", StoryID: "s1", Type: "RETCON", Status: "ACTIVE"},
		{ID: "official-active", StoryID: "s1", Type: "OFFICIAL", Status: "ACTIVE"},
	}
	svc := NewService(store)

	branch, err := svc.GetActiveOfficialCanonBranch(context.Background(), "s1")
	if err != nil {
		t.Fatalf("get active official branch: %v", err)
	}
	if branch.ID != "official-active" {
		t.Fatalf("expected official-active, got %q", branch.ID)
	}
}

func TestGetActiveOfficialCanonBranchFailsClosedWhenMissing(t *testing.T) {
	store := newCanonFakeStore()
	store.branches["s1"] = []CanonBranch{
		{ID: "retcon-active", StoryID: "s1", Type: "RETCON", Status: "ACTIVE"},
		{ID: "official-archived", StoryID: "s1", Type: "OFFICIAL", Status: "ARCHIVED"},
	}
	svc := NewService(store)

	_, err := svc.GetActiveOfficialCanonBranch(context.Background(), "s1")
	if !errors.Is(err, ErrCanonBranchNotFound) {
		t.Fatalf("expected ErrCanonBranchNotFound, got %v", err)
	}
}

func TestGetActiveOfficialCanonBranchFailsClosedWhenAmbiguous(t *testing.T) {
	store := newCanonFakeStore()
	store.branches["s1"] = []CanonBranch{
		{ID: "official-active-1", StoryID: "s1", Type: "OFFICIAL", Status: "ACTIVE"},
		{ID: "official-active-2", StoryID: "s1", Type: "OFFICIAL", Status: "ACTIVE"},
	}
	svc := NewService(store)

	_, err := svc.GetActiveOfficialCanonBranch(context.Background(), "s1")
	if !errors.Is(err, ErrCanonBranchNotFound) {
		t.Fatalf("expected ambiguous authority to fail closed, got %v", err)
	}
}

func TestGetActiveOfficialCanonBranchRejectsEmptyStoryID(t *testing.T) {
	store := newCanonFakeStore()
	svc := NewService(store)

	_, err := svc.GetActiveOfficialCanonBranch(context.Background(), "")
	if !errors.Is(err, ErrCanonBranchNotFound) {
		t.Fatalf("expected ErrCanonBranchNotFound, got %v", err)
	}
}
