package planning

import (
	"context"
	"testing"
)

func TestGetContextSnapshot(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	sn, _ := svc.CreateContextSnapshot(context.Background(), "s1", "ch1", "b1", "e1", "a1", "p1", "pv1", "wv1", "mock", "mock-model")

	got, err := svc.GetContextSnapshot(context.Background(), sn.ID)
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if got.ID != sn.ID {
		t.Fatalf("expected %q, got %q", sn.ID, got.ID)
	}
	if got.Provider != "mock" {
		t.Fatalf("expected provider mock, got %q", got.Provider)
	}
}

func TestGetContextSnapshotNotFound(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.GetContextSnapshot(context.Background(), "missing"); err == nil {
		t.Fatal("expected error for missing snapshot")
	}
}

func TestAnalyzeThreadInactivity(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	store.threads["s1"] = []PlotThread{
		{ID: "t1", StoryID: "s1", Title: "Thread A", Status: "OPEN", Importance: "MAJOR"},
		{ID: "t2", StoryID: "s1", Title: "Thread B", Status: "OPEN", Importance: "NORMAL"},
	}
	store.events["t1"] = []PlotThreadEvent{
		{ID: "e1", PlotThreadID: "t1", ChapterID: "ch1", EventType: "ADVANCE"},
	}

	res, err := svc.AnalyzeThreadInactivity(context.Background(), "s1", 1)
	if err != nil {
		t.Fatalf("analyze inactivity: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 inactive thread, got %d", len(res))
	}
	if res[0].ThreadID != "t2" {
		t.Fatalf("expected t2 inactive, got %q", res[0].ThreadID)
	}
}
