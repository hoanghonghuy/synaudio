package generation

import (
	"context"
	"testing"
)

func TestStartBatchGenerationCreatesSequentialJobs(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	chapterIDs := []string{"ch1", "ch2", "ch3"}

	run, err := svc.StartBatchGeneration(context.Background(), "s1", chapterIDs, "u1")
	if err != nil {
		t.Fatalf("start batch: %v", err)
	}
	if run.RunType != "CHAPTER_GENERATION" {
		t.Fatalf("expected CHAPTER_GENERATION, got %q", run.RunType)
	}
	if run.Status != "PENDING" {
		t.Fatalf("expected PENDING, got %q", run.Status)
	}

	jobs := store.jobs[run.ID]
	if len(jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(jobs))
	}
	for i, j := range jobs {
		if j.JobType != "WRITER" {
			t.Fatalf("expected WRITER job, got %q", j.JobType)
		}
		if j.OutputRef["chapter_id"] != chapterIDs[i] {
			t.Fatalf("expected chapter %q, got %q", chapterIDs[i], j.OutputRef["chapter_id"])
		}
	}
}

func TestStartBatchGenerationRejectsEmptyChapters(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.StartBatchGeneration(context.Background(), "s1", nil, "u1"); err == nil {
		t.Fatal("expected error for empty chapters")
	}
}

func TestMarkDownstreamStale(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	run, _ := svc.StartBatchGeneration(context.Background(), "s1", []string{"ch1", "ch2", "ch3"}, "u1")

	stale, err := svc.MarkDownstreamStale(context.Background(), run.ID, "ch2")
	if err != nil {
		t.Fatalf("mark stale: %v", err)
	}
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale job, got %d", len(stale))
	}
	if stale[0].OutputRef["chapter_id"] != "ch3" {
		t.Fatalf("expected ch3 stale, got %q", stale[0].OutputRef["chapter_id"])
	}
	if stale[0].Status != "STALE" {
		t.Fatalf("expected STALE, got %q", stale[0].Status)
	}
}
