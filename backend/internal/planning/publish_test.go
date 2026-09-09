package planning

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type publishFakeStore struct {
	*fakeStore
	statuses                map[string]string
	publishMu               sync.Mutex
	onDuringPublishValidate func()
}

var _ atomicPublishStore = (*publishFakeStore)(nil)

func newPublishFakeStore() *publishFakeStore {
	return &publishFakeStore{
		fakeStore: newFakeStore(),
		statuses:  map[string]string{},
	}
}

func (s *publishFakeStore) UpdateChapterStatus(ctx context.Context, chapterID, status string) (Chapter, error) {
	for storyID, cs := range s.chapters {
		for i, c := range cs {
			if c.ID == chapterID {
				c.Status = status
				s.chapters[storyID][i] = c
				s.statuses[chapterID] = status
				return c, nil
			}
		}
	}
	return Chapter{}, ErrChapterNotFound
}

func (s *publishFakeStore) GetChapter(ctx context.Context, chapterID string) (Chapter, error) {
	for _, cs := range s.chapters {
		for _, c := range cs {
			if c.ID == chapterID {
				return c, nil
			}
		}
	}
	return Chapter{}, ErrChapterNotFound
}

func (s *publishFakeStore) PublishChapterAtomically(
	ctx context.Context,
	chapterID string,
	validate ChapterPublishValidator,
) (Chapter, error) {
	s.publishMu.Lock()
	defer s.publishMu.Unlock()

	ch, err := s.GetChapter(ctx, chapterID)
	if err != nil {
		return Chapter{}, err
	}
	if ch.Status != "READY" {
		return Chapter{}, ErrPublishNotReady
	}

	if s.onDuringPublishValidate != nil {
		s.onDuringPublishValidate()
	}

	missing, err := validate(ctx)
	if err != nil {
		return Chapter{}, err
	}
	if len(missing) > 0 {
		return Chapter{}, ErrPublishNotReady
	}

	return s.UpdateChapterStatus(ctx, chapterID, "PUBLISHED")
}

type fakePublishChecker struct {
	missing []string
}

func (c *fakePublishChecker) CheckPublishReady(ctx context.Context, chapterID string) ([]string, error) {
	return c.missing, nil
}

func TestPublishChapterTransitionsReadyToPublished(t *testing.T) {
	store := newPublishFakeStore()
	svc := NewService(store, WithPublishChecker(&fakePublishChecker{}))

	ch, _ := svc.CreateChapter(context.Background(), "s1", "Chapter 1", "u1")
	// Manually set to READY.
	_, _ = store.UpdateChapterStatus(context.Background(), ch.ID, "READY")

	published, err := svc.PublishChapter(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if published.Status != "PUBLISHED" {
		t.Fatalf("expected PUBLISHED, got %q", published.Status)
	}
}

func TestPublishChapterRejectsWhenNotReady(t *testing.T) {
	store := newPublishFakeStore()
	svc := NewService(store, WithPublishChecker(&fakePublishChecker{missing: []string{"audio"}}))

	ch, _ := svc.CreateChapter(context.Background(), "s1", "Chapter 1", "u1")
	_, _ = store.UpdateChapterStatus(context.Background(), ch.ID, "READY")

	if _, err := svc.PublishChapter(context.Background(), ch.ID); !errors.Is(err, ErrPublishNotReady) {
		t.Fatalf("expected ErrPublishNotReady, got %v", err)
	}
}

func TestPublishChapterRejectsWhenNotInReadyState(t *testing.T) {
	store := newPublishFakeStore()
	svc := NewService(store, WithPublishChecker(&fakePublishChecker{}))

	ch, _ := svc.CreateChapter(context.Background(), "s1", "Chapter 1", "u1")
	// Still DRAFT.

	if _, err := svc.PublishChapter(context.Background(), ch.ID); !errors.Is(err, ErrPublishNotReady) {
		t.Fatalf("expected ErrPublishNotReady, got %v", err)
	}
}

func TestUnpublishChapterTransitionsPublishedToReady(t *testing.T) {
	store := newPublishFakeStore()
	svc := NewService(store, WithPublishChecker(&fakePublishChecker{}))

	ch, _ := svc.CreateChapter(context.Background(), "s1", "Chapter 1", "u1")
	_, _ = store.UpdateChapterStatus(context.Background(), ch.ID, "PUBLISHED")

	unpublished, err := svc.UnpublishChapter(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("unpublish: %v", err)
	}
	if unpublished.Status != "READY" {
		t.Fatalf("expected READY, got %q", unpublished.Status)
	}
}

func TestUnpublishChapterRejectsWhenNotPublished(t *testing.T) {
	store := newPublishFakeStore()
	svc := NewService(store, WithPublishChecker(&fakePublishChecker{}))

	ch, _ := svc.CreateChapter(context.Background(), "s1", "Chapter 1", "u1")

	if _, err := svc.UnpublishChapter(context.Background(), ch.ID); !errors.Is(err, ErrNotPublished) {
		t.Fatalf("expected ErrNotPublished, got %v", err)
	}
}

func TestPublishChapterRejectsWhenPublishCheckerMissing(t *testing.T) {
	store := newPublishFakeStore()
	svc := NewService(store)

	ch, _ := svc.CreateChapter(context.Background(), "s1", "Chapter 1", "u1")
	_, _ = store.UpdateChapterStatus(context.Background(), ch.ID, "READY")

	if _, err := svc.PublishChapter(context.Background(), ch.ID); !errors.Is(err, ErrPublishAuthorityRequired) {
		t.Fatalf("expected ErrPublishAuthorityRequired, got %v", err)
	}
}

func TestMarkChapterReadyTransitionsWhenGatesPass(t *testing.T) {
	store := newPublishFakeStore()
	svc := NewService(store, WithPublishChecker(&fakePublishChecker{}))

	ch, _ := svc.CreateChapter(context.Background(), "s1", "Chapter 1", "u1")
	if ch.Status != "DRAFT" {
		t.Fatalf("expected DRAFT, got %q", ch.Status)
	}

	ready, err := svc.MarkChapterReady(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("mark ready: %v", err)
	}
	if ready.Status != "READY" {
		t.Fatalf("expected READY, got %q", ready.Status)
	}
}

func TestMarkChapterReadyRejectsWhenGatesMissing(t *testing.T) {
	store := newPublishFakeStore()
	svc := NewService(store, WithPublishChecker(&fakePublishChecker{missing: []string{"active_audio"}}))

	ch, _ := svc.CreateChapter(context.Background(), "s1", "Chapter 1", "u1")

	if _, err := svc.MarkChapterReady(context.Background(), ch.ID); !errors.Is(err, ErrPublishNotReady) {
		t.Fatalf("expected ErrPublishNotReady, got %v", err)
	}
	if store.statuses[ch.ID] == "READY" {
		t.Fatal("chapter must not transition to READY when gates fail")
	}
}

func TestMarkChapterReadyIsIdempotentWhenAlreadyReady(t *testing.T) {
	store := newPublishFakeStore()
	svc := NewService(store, WithPublishChecker(&fakePublishChecker{}))

	ch, _ := svc.CreateChapter(context.Background(), "s1", "Chapter 1", "u1")
	_, _ = store.UpdateChapterStatus(context.Background(), ch.ID, "READY")

	ready, err := svc.MarkChapterReady(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("mark ready: %v", err)
	}
	if ready.Status != "READY" {
		t.Fatalf("expected READY, got %q", ready.Status)
	}
}

func TestMarkChapterReadyRejectsPublishedChapter(t *testing.T) {
	store := newPublishFakeStore()
	svc := NewService(store, WithPublishChecker(&fakePublishChecker{}))

	ch, _ := svc.CreateChapter(context.Background(), "s1", "Chapter 1", "u1")
	_, _ = store.UpdateChapterStatus(context.Background(), ch.ID, "PUBLISHED")

	if _, err := svc.MarkChapterReady(context.Background(), ch.ID); !errors.Is(err, ErrChapterNotMarkable) {
		t.Fatalf("expected ErrChapterNotMarkable, got %v", err)
	}
}

func TestPublishChapterRejectsDraftWithoutMarkReady(t *testing.T) {
	store := newPublishFakeStore()
	svc := NewService(store, WithPublishChecker(&fakePublishChecker{}))

	ch, _ := svc.CreateChapter(context.Background(), "s1", "Chapter 1", "u1")

	if _, err := svc.PublishChapter(context.Background(), ch.ID); !errors.Is(err, ErrPublishNotReady) {
		t.Fatalf("expected ErrPublishNotReady without mark-ready, got %v", err)
	}
}
