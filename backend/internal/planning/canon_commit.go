package planning

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// MemoryExtractor extracts canonical facts from approved content.
type MemoryExtractor interface {
	ExtractMemory(ctx context.Context, in MemoryExtractionInput) (MemoryExtraction, error)
}

// MemoryExtractionInput is the input for memory extraction.
type MemoryExtractionInput struct {
	StoryID         string
	ChapterID       string
	ContentRevisionID string
	ContentText     string
}

// ExtractedFact is a fact extracted from content.
type ExtractedFact struct {
	SubjectType string
	SubjectID   string
	FactType    string
	Value       map[string]any
}

// MemoryExtraction is the result of memory extraction.
type MemoryExtraction struct {
	Facts []ExtractedFact
}

// CanonChangeItem is a single change in a canon version.
type CanonChangeItem struct {
	ID             string
	CanonVersionID string
	EntityType     string
	EntityID       string
	ChangeType     string
	Metadata       map[string]any
}

// CanonCommitResult is the result of a canon commit.
type CanonCommitResult struct {
	Version     CanonVersion
	ChangeItems []CanonChangeItem
}

// CommitCanon performs Memory Extraction and commits a new OFFICIAL canon version.
func (s *Service) CommitCanon(ctx context.Context, storyID, branchID, sourceChapterID, contentRevisionID, committedBy string) (CanonCommitResult, error) {
	if s.memoryExtractor == nil {
		return CanonCommitResult{}, errors.New("memory extractor not configured")
	}

	extraction, err := s.memoryExtractor.ExtractMemory(ctx, MemoryExtractionInput{
		StoryID:           storyID,
		ChapterID:         sourceChapterID,
		ContentRevisionID: contentRevisionID,
	})
	if err != nil {
		return CanonCommitResult{}, err
	}

	seq, err := s.store.NextCanonSequence(ctx, branchID)
	if err != nil {
		return CanonCommitResult{}, err
	}

	version := CanonVersion{
		ID:              uuid.NewString(),
		StoryID:         storyID,
		BranchID:        branchID,
		SequenceNo:      seq,
		SourceChapterID: sourceChapterID,
		Status:          "OFFICIAL",
		CommittedBy:     committedBy,
	}

	created, err := s.store.CreateCanonVersion(ctx, version)
	if err != nil {
		return CanonCommitResult{}, err
	}

	res := CanonCommitResult{Version: created}

	for _, f := range extraction.Facts {
		item := CanonChangeItem{
			ID:             uuid.NewString(),
			CanonVersionID: created.ID,
			EntityType:     f.SubjectType,
			EntityID:       f.SubjectID,
			ChangeType:     "UPSERT",
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
