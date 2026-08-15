package planning

import (
	"context"
	"testing"
)

type threadFakeStore struct {
	*fakeStore
	threads map[string][]PlotThread
	events  map[string][]PlotThreadEvent
}

func newThreadFakeStore() *threadFakeStore {
	return &threadFakeStore{
		fakeStore: newFakeStore(),
		threads:   map[string][]PlotThread{},
		events:    map[string][]PlotThreadEvent{},
	}
}

func (s *threadFakeStore) CreatePlotThread(ctx context.Context, t PlotThread) (PlotThread, error) {
	s.threads[t.StoryID] = append(s.threads[t.StoryID], t)
	return t, nil
}

func (s *threadFakeStore) ListPlotThreads(ctx context.Context, storyID string) ([]PlotThread, error) {
	return s.threads[storyID], nil
}

func (s *threadFakeStore) CreatePlotThreadEvent(ctx context.Context, e PlotThreadEvent) (PlotThreadEvent, error) {
	s.events[e.PlotThreadID] = append(s.events[e.PlotThreadID], e)
	return e, nil
}

func TestCreatePlotThread(t *testing.T) {
	store := newThreadFakeStore()
	svc := NewService(store)

	th, err := svc.CreatePlotThread(context.Background(), "s1", "The Lost Crown", "A crown is missing", "MAJOR")
	if err != nil {
		t.Fatalf("create thread: %v", err)
	}
	if th.Status != "OPEN" {
		t.Fatalf("expected OPEN, got %q", th.Status)
	}
	if th.Importance != "MAJOR" {
		t.Fatalf("expected MAJOR, got %q", th.Importance)
	}
}

func TestCreatePlotThreadRejectsEmptyTitle(t *testing.T) {
	store := newThreadFakeStore()
	svc := NewService(store)

	if _, err := svc.CreatePlotThread(context.Background(), "s1", "  ", "summary", "MAJOR"); err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestCreatePlotThreadEvent(t *testing.T) {
	store := newThreadFakeStore()
	svc := NewService(store)

	th, _ := svc.CreatePlotThread(context.Background(), "s1", "The Lost Crown", "summary", "MAJOR")

	e, err := svc.CreatePlotThreadEvent(context.Background(), th.ID, "ADVANCED", "c1", map[string]any{"note": "found clue"})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	if e.EventType != "ADVANCED" {
		t.Fatalf("expected ADVANCED, got %q", e.EventType)
	}
}

func TestListPlotThreadsReturnsAll(t *testing.T) {
	store := newThreadFakeStore()
	svc := NewService(store)

	_, _ = svc.CreatePlotThread(context.Background(), "s1", "Thread A", "summary", "MAJOR")
	_, _ = svc.CreatePlotThread(context.Background(), "s1", "Thread B", "summary", "MINOR")

	threads, err := svc.ListPlotThreads(context.Background(), "s1")
	if err != nil {
		t.Fatalf("list threads: %v", err)
	}
	if len(threads) != 2 {
		t.Fatalf("expected 2 threads, got %d", len(threads))
	}
}
