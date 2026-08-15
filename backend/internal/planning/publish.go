package planning

import (
	"context"
	"errors"
)

var (
	ErrPublishNotReady = errors.New("chapter not ready to publish")
	ErrNotPublished    = errors.New("chapter not published")
)

// PublishChecker reports missing dependencies blocking chapter publish.
type PublishChecker interface {
	CheckPublishReady(ctx context.Context, chapterID string) (missing []string, err error)
}

// PublishChapter transitions a READY chapter to PUBLISHED.
func (s *Service) PublishChapter(ctx context.Context, chapterID string) (Chapter, error) {
	ch, err := s.store.GetChapter(ctx, chapterID)
	if err != nil {
		return Chapter{}, err
	}
	if ch.Status != "READY" {
		return Chapter{}, ErrPublishNotReady
	}

	if s.publishChecker != nil {
		missing, err := s.publishChecker.CheckPublishReady(ctx, chapterID)
		if err != nil {
			return Chapter{}, err
		}
		if len(missing) > 0 {
			return Chapter{}, ErrPublishNotReady
		}
	}

	return s.store.UpdateChapterStatus(ctx, chapterID, "PUBLISHED")
}

// UnpublishChapter transitions a PUBLISHED chapter back to READY.
func (s *Service) UnpublishChapter(ctx context.Context, chapterID string) (Chapter, error) {
	ch, err := s.store.GetChapter(ctx, chapterID)
	if err != nil {
		return Chapter{}, err
	}
	if ch.Status != "PUBLISHED" {
		return Chapter{}, ErrNotPublished
	}

	return s.store.UpdateChapterStatus(ctx, chapterID, "READY")
}
