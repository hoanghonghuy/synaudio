package planning

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrCanonVersionNotFound = errors.New("canon version not found")
)

// CreateProvisionalCanonVersion creates a PROVISIONAL canon version in a
// PROVISIONAL branch. Provisional canon never serves public listener requests.
func (s *Service) CreateProvisionalCanonVersion(ctx context.Context, storyID, branchID, sourceChapterID, sourceContentRevisionID string) (CanonVersion, error) {
	seq, err := s.store.NextCanonSequence(ctx, branchID)
	if err != nil {
		return CanonVersion{}, err
	}

	v := CanonVersion{
		ID:                     uuid.NewString(),
		StoryID:                storyID,
		BranchID:               branchID,
		SequenceNo:             seq,
		SourceChapterID:        sourceChapterID,
		SourceContentRevisionID: sourceContentRevisionID,
		Status:                 "PROVISIONAL",
	}

	return s.store.CreateCanonVersion(ctx, v)
}

// PromoteProvisionalVersion promotes a PROVISIONAL canon version to OFFICIAL.
// This is the atomic promotion step that makes provisional canon official.
func (s *Service) PromoteProvisionalVersion(ctx context.Context, versionID, committedBy string) (CanonVersion, error) {
	v, err := s.store.GetCanonVersion(ctx, versionID)
	if err != nil {
		return CanonVersion{}, err
	}

	if v.Status != "PROVISIONAL" {
		return CanonVersion{}, errors.New("only PROVISIONAL versions can be promoted")
	}

	v.Status = "OFFICIAL"
	v.CommittedBy = committedBy

	return s.store.UpdateCanonVersion(ctx, v)
}
