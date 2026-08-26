package planning

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// RepairCanonInput is the input for repairing canonical data.
type RepairCanonInput struct {
	StoryID           string
	BranchID          string
	SourceChapterID   string
	ContentRevisionID string
	CommittedBy       string
}

// RepairCanonData re-extracts structured memory from published evidence and
// commits a new OFFICIAL canon version, superseding any conflicting facts.
func (s *Service) RepairCanonData(ctx context.Context, in RepairCanonInput) (CanonCommitResult, error) {
	if s.memoryExtractor == nil {
		return CanonCommitResult{}, errors.New("memory extractor not configured")
	}

	extraction, err := s.memoryExtractor.ExtractMemory(ctx, MemoryExtractionInput{
		StoryID:           in.StoryID,
		ChapterID:         in.SourceChapterID,
		ContentRevisionID: in.ContentRevisionID,
	})
	if err != nil {
		return CanonCommitResult{}, err
	}

	seq, err := s.store.NextCanonSequence(ctx, in.BranchID)
	if err != nil {
		return CanonCommitResult{}, err
	}

	version := CanonVersion{
		ID:              uuid.NewString(),
		StoryID:         in.StoryID,
		BranchID:        in.BranchID,
		SequenceNo:      seq,
		SourceChapterID: in.SourceChapterID,
		Status:          "OFFICIAL",
		CommittedBy:     in.CommittedBy,
	}

	created, err := s.store.CreateCanonVersion(ctx, version)
	if err != nil {
		return CanonCommitResult{}, err
	}

	res := CanonCommitResult{Version: created}

	for _, f := range extraction.Facts {
		// Supersede any existing ACTIVE fact with the same subject + fact type.
		existing, err := s.store.ListFacts(ctx, in.StoryID)
		if err != nil {
			return CanonCommitResult{}, err
		}
		for _, old := range existing {
			if old.Status == "ACTIVE" && old.SubjectType == f.SubjectType && old.SubjectID == f.SubjectID && old.FactType == f.FactType {
				old.Status = "SUPERSEDED"
				if _, err := s.store.UpdateFact(ctx, old); err != nil {
					return CanonCommitResult{}, err
				}
			}
		}

		item := CanonChangeItem{
			ID:             uuid.NewString(),
			CanonVersionID: created.ID,
			EntityType:     f.SubjectType,
			EntityID:       f.SubjectID,
			ChangeType:     "REPAIR",
			Metadata:       map[string]any{"fact_type": f.FactType, "value": f.Value},
		}

		createdItem, err := s.store.CreateCanonChangeItem(ctx, item)
		if err != nil {
			return CanonCommitResult{}, err
		}
		res.ChangeItems = append(res.ChangeItems, createdItem)
	}

	return res, nil
}
