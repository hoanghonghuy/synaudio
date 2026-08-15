package generation

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// EditContent creates a new ADMIN_EDIT revision based on an existing one.
func (s *Service) EditContent(ctx context.Context, chapterID, basedOnRevisionID, newText, editedBy string) (ContentRevision, error) {
	newText = strings.TrimSpace(newText)
	if newText == "" {
		return ContentRevision{}, errors.New("content text must not be empty")
	}

	if _, err := s.store.GetContentRevision(ctx, basedOnRevisionID); err != nil {
		return ContentRevision{}, err
	}

	revisionNo, err := s.store.NextContentRevision(ctx, chapterID)
	if err != nil {
		return ContentRevision{}, err
	}

	r := ContentRevision{
		ID:                uuid.NewString(),
		ChapterID:         chapterID,
		RevisionNo:        revisionNo,
		ContentText:       newText,
		SourceType:        "ADMIN_EDIT",
		BasedOnRevisionID: basedOnRevisionID,
		Status:            "CANDIDATE",
		CreatedBy:         editedBy,
	}

	return s.store.CreateContentRevision(ctx, r)
}

// RegenerateContent produces a fresh AI_GENERATED candidate for a chapter.
func (s *Service) RegenerateContent(ctx context.Context, chapterID, basedOnRevisionID, requestedBy string) (ContentRevision, error) {
	if s.textAI == nil {
		return ContentRevision{}, errors.New("text AI not configured")
	}

	base, err := s.store.GetContentRevision(ctx, basedOnRevisionID)
	if err != nil {
		return ContentRevision{}, err
	}

	out, err := s.textAI.GenerateText(ctx, TextAIInput{Prompt: base.ContentText})
	if err != nil {
		return ContentRevision{}, err
	}

	revisionNo, err := s.store.NextContentRevision(ctx, chapterID)
	if err != nil {
		return ContentRevision{}, err
	}

	r := ContentRevision{
		ID:          uuid.NewString(),
		ChapterID:   chapterID,
		RevisionNo:  revisionNo,
		ContentText: strings.TrimSpace(out.Text),
		SourceType:  "AI_GENERATED",
		Status:      "CANDIDATE",
		CreatedBy:   requestedBy,
	}

	return s.store.CreateContentRevision(ctx, r)
}

// RejectContent marks a content revision as REJECTED.
func (s *Service) RejectContent(ctx context.Context, revisionID, rejectedBy, reason string) (ContentRevision, error) {
	if _, err := s.store.GetContentRevision(ctx, revisionID); err != nil {
		return ContentRevision{}, err
	}

	return s.store.UpdateContentRevisionStatus(ctx, revisionID, "REJECTED")
}
