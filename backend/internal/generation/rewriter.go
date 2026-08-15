package generation

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// RewriteChapter produces a new AI_REWRITE revision based on an existing one.
func (s *Service) RewriteChapter(ctx context.Context, chapterID, basedOnRevisionID, feedback, createdBy string) (ContentRevision, error) {
	if s.textAI == nil {
		return ContentRevision{}, errors.New("text AI not configured")
	}

	base, err := s.store.GetContentRevision(ctx, basedOnRevisionID)
	if err != nil {
		return ContentRevision{}, err
	}

	prompt := "Rewrite the following chapter text. Feedback: " + feedback + "\n\n" + base.ContentText
	out, err := s.textAI.GenerateText(ctx, TextAIInput{Prompt: prompt})
	if err != nil {
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
		ContentText:       strings.TrimSpace(out.Text),
		SourceType:        "AI_REWRITE",
		BasedOnRevisionID: basedOnRevisionID,
		Status:            "CANDIDATE",
		CreatedBy:         createdBy,
	}

	return s.store.CreateContentRevision(ctx, r)
}
