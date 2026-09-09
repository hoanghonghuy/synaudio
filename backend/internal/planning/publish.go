package planning

import (
	"context"
	"errors"
)

var (
	ErrPublishNotReady          = errors.New("chapter not ready to publish")
	ErrPublishAuthorityRequired = errors.New("publish authority not configured")
	ErrNotPublished             = errors.New("chapter not published")
	ErrChapterNotMarkable       = errors.New("chapter cannot be marked ready")
)

// PublishChecker reports missing dependencies blocking chapter publish.
type PublishChecker interface {
	CheckPublishReady(ctx context.Context, chapterID string) (missing []string, err error)
}

// ChapterPublishValidator revalidates publish prerequisites inside the atomic
// publish boundary.
type ChapterPublishValidator func(context.Context) ([]string, error)

// atomicPublishStore serializes publish readiness revalidation with narration
// version allocation on the same chapter-scoped advisory lock.
type atomicPublishStore interface {
	PublishChapterAtomically(ctx context.Context, chapterID string, validate ChapterPublishValidator) (Chapter, error)
}

// MarkChapterReady transitions a chapter into READY when authoritative publish
// prerequisites are satisfied. READY means publishable right now.
func (s *Service) MarkChapterReady(ctx context.Context, chapterID string) (Chapter, error) {
	if s.publishChecker == nil {
		return Chapter{}, ErrPublishAuthorityRequired
	}

	ch, err := s.store.GetChapter(ctx, chapterID)
	if err != nil {
		return Chapter{}, err
	}

	switch ch.Status {
	case "READY":
		return ch, nil
	case "PUBLISHED", "ARCHIVED":
		return Chapter{}, ErrChapterNotMarkable
	}

	missing, err := s.publishChecker.CheckPublishReady(ctx, chapterID)
	if err != nil {
		return Chapter{}, err
	}
	if len(missing) > 0 {
		return Chapter{}, ErrPublishNotReady
	}

	return s.store.UpdateChapterStatus(ctx, chapterID, "READY")
}

// PublishChapter transitions a READY chapter to PUBLISHED.
func (s *Service) PublishChapter(ctx context.Context, chapterID string) (Chapter, error) {
	if s.publishChecker == nil {
		return Chapter{}, ErrPublishAuthorityRequired
	}

	validate := func(ctx context.Context) ([]string, error) {
		return s.publishChecker.CheckPublishReady(ctx, chapterID)
	}

	if atomic, ok := s.store.(atomicPublishStore); ok {
		return atomic.PublishChapterAtomically(ctx, chapterID, validate)
	}

	ch, err := s.store.GetChapter(ctx, chapterID)
	if err != nil {
		return Chapter{}, err
	}
	if ch.Status != "READY" {
		return Chapter{}, ErrPublishNotReady
	}

	missing, err := validate(ctx)
	if err != nil {
		return Chapter{}, err
	}
	if len(missing) > 0 {
		return Chapter{}, ErrPublishNotReady
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
