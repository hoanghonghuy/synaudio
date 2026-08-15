package planning

import (
	"context"

	"github.com/google/uuid"
)

// CanonBranch is a lineage of canon versions.
type CanonBranch struct {
	ID      string
	StoryID string
	Type    string
	Status  string
}

// CanonVersion is a committed point in a canon branch.
type CanonVersion struct {
	ID             string
	StoryID        string
	BranchID       string
	SequenceNo     int
	ParentVersionID string
	SourceChapterID string
	Status         string
	CommittedBy    string
}

// CreateCanonBranch creates a new ACTIVE canon branch.
func (s *Service) CreateCanonBranch(ctx context.Context, storyID, branchType string) (CanonBranch, error) {
	if branchType == "" {
		branchType = "OFFICIAL"
	}

	b := CanonBranch{
		ID:      uuid.NewString(),
		StoryID: storyID,
		Type:    branchType,
		Status:  "ACTIVE",
	}

	return s.store.CreateCanonBranch(ctx, b)
}

// CreateCanonVersion creates a new canon version with the next sequence number.
func (s *Service) CreateCanonVersion(ctx context.Context, storyID, branchID, sourceChapterID, committedBy string) (CanonVersion, error) {
	seq, err := s.store.NextCanonSequence(ctx, branchID)
	if err != nil {
		return CanonVersion{}, err
	}

	v := CanonVersion{
		ID:              uuid.NewString(),
		StoryID:         storyID,
		BranchID:        branchID,
		SequenceNo:      seq,
		SourceChapterID: sourceChapterID,
		Status:          "DRAFT",
		CommittedBy:     committedBy,
	}

	return s.store.CreateCanonVersion(ctx, v)
}

// ListCanonVersions returns all versions in a branch, ordered by sequence.
func (s *Service) ListCanonVersions(ctx context.Context, branchID string) ([]CanonVersion, error) {
	return s.store.ListCanonVersions(ctx, branchID)
}
