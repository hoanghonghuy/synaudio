package planning

import (
	"context"
	"errors"
	"testing"
)

type chapterFakeStore struct {
	*fakeStore
	chapters    map[string][]Chapter
	plans       map[string][]ChapterPlanRevision
	nextPlan    map[string]int
	nextChapter map[string]int
}

func newChapterFakeStore() *chapterFakeStore {
	return &chapterFakeStore{
		fakeStore:   newFakeStore(),
		chapters:    map[string][]Chapter{},
		plans:       map[string][]ChapterPlanRevision{},
		nextPlan:    map[string]int{},
		nextChapter: map[string]int{},
	}
}

func (s *chapterFakeStore) NextChapterNumber(ctx context.Context, storyID string) (int, error) {
	s.nextChapter[storyID]++
	return s.nextChapter[storyID], nil
}

func (s *chapterFakeStore) CreateChapter(ctx context.Context, c Chapter) (Chapter, error) {
	s.chapters[c.StoryID] = append(s.chapters[c.StoryID], c)
	return c, nil
}

func (s *chapterFakeStore) NextPlanRevision(ctx context.Context, chapterID string) (int, error) {
	s.nextPlan[chapterID]++
	return s.nextPlan[chapterID], nil
}

func (s *chapterFakeStore) CreatePlanRevision(ctx context.Context, p ChapterPlanRevision) (ChapterPlanRevision, error) {
	s.plans[p.ChapterID] = append(s.plans[p.ChapterID], p)
	return p, nil
}

func (s *chapterFakeStore) GetChapter(ctx context.Context, chapterID string) (Chapter, error) {
	for _, cs := range s.chapters {
		for _, c := range cs {
			if c.ID == chapterID {
				return c, nil
			}
		}
	}
	return Chapter{}, ErrChapterNotFound
}

func (s *chapterFakeStore) ListChapters(ctx context.Context, storyID string) ([]Chapter, error) {
	return s.chapters[storyID], nil
}

func TestCreateChapterAssignsSequentialNumber(t *testing.T) {
	store := newChapterFakeStore()
	svc := NewService(store)

	c1, err := svc.CreateChapter(context.Background(), "s1", "Chapter One", "u1")
	if err != nil {
		t.Fatalf("create chapter 1: %v", err)
	}
	if c1.ChapterNumber != 1 {
		t.Fatalf("expected number 1, got %d", c1.ChapterNumber)
	}

	c2, err := svc.CreateChapter(context.Background(), "s1", "Chapter Two", "u1")
	if err != nil {
		t.Fatalf("create chapter 2: %v", err)
	}
	if c2.ChapterNumber != 2 {
		t.Fatalf("expected number 2, got %d", c2.ChapterNumber)
	}
}

func TestCreateChapterRejectsEmptyTitle(t *testing.T) {
	store := newChapterFakeStore()
	svc := NewService(store)

	if _, err := svc.CreateChapter(context.Background(), "s1", "  ", "u1"); err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestCreatePlanRevisionAssignsSequentialRevision(t *testing.T) {
	store := newChapterFakeStore()
	svc := NewService(store)

	c, _ := svc.CreateChapter(context.Background(), "s1", "Chapter One", "u1")

	p1, err := svc.CreatePlanRevision(context.Background(), c.ID, map[string]any{"objective": "A"}, "u1")
	if err != nil {
		t.Fatalf("create plan 1: %v", err)
	}
	if p1.RevisionNo != 1 {
		t.Fatalf("expected revision 1, got %d", p1.RevisionNo)
	}

	p2, err := svc.CreatePlanRevision(context.Background(), c.ID, map[string]any{"objective": "B"}, "u1")
	if err != nil {
		t.Fatalf("create plan 2: %v", err)
	}
	if p2.RevisionNo != 2 {
		t.Fatalf("expected revision 2, got %d", p2.RevisionNo)
	}
}

func TestListChaptersReturnsAll(t *testing.T) {
	store := newChapterFakeStore()
	svc := NewService(store)

	_, _ = svc.CreateChapter(context.Background(), "s1", "Chapter One", "u1")
	_, _ = svc.CreateChapter(context.Background(), "s1", "Chapter Two", "u1")

	chapters, err := svc.ListChapters(context.Background(), "s1")
	if err != nil {
		t.Fatalf("list chapters: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}
}

func TestGetChapterReturnsNotFound(t *testing.T) {
	store := newChapterFakeStore()
	svc := NewService(store)

	if _, err := svc.GetChapter(context.Background(), "missing"); !errors.Is(err, ErrChapterNotFound) {
		t.Fatalf("expected ErrChapterNotFound, got %v", err)
	}
}
