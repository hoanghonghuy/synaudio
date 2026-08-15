package generation

import (
	"context"
	"errors"
)

// Writer produces Chapter prose from a Chapter Plan using a TextAI provider.
type Writer struct {
	textAI TextAIProvider
}

// WriteChapter generates prose and persists it as a new CANDIDATE content revision.
func (s *Service) WriteChapter(ctx context.Context, chapterID, prompt, createdBy string) (ContentRevision, error) {
	if s.textAI == nil {
		return ContentRevision{}, errors.New("text AI not configured")
	}

	out, err := s.textAI.GenerateText(ctx, TextAIInput{Prompt: prompt})
	if err != nil {
		return ContentRevision{}, err
	}

	return s.CreateContentRevision(ctx, chapterID, out.Text, "AI_GENERATED", createdBy)
}
