package generation

import (
	"context"
	"testing"
)

func TestRecordUsage(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "ch1", "u1")
	job, _ := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)
	attempt, _ := svc.CreateJobAttempt(context.Background(), job.ID, "mock", "mock-model")

	usage := map[string]any{"input_tokens": 100, "output_tokens": 200, "cost_usd": 0.01}

	updated, err := svc.RecordUsage(context.Background(), attempt.ID, usage)
	if err != nil {
		t.Fatalf("record usage: %v", err)
	}
	if updated.Usage["input_tokens"] != 100 {
		t.Fatalf("expected input_tokens 100, got %v", updated.Usage["input_tokens"])
	}
}

func TestListUsageByStory(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "ch1", "u1")
	job, _ := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)
	attempt, _ := svc.CreateJobAttempt(context.Background(), job.ID, "mock", "mock-model")

	_, _ = svc.RecordUsage(context.Background(), attempt.ID, map[string]any{"input_tokens": 50, "output_tokens": 60})

	usage, err := svc.ListUsageByStory(context.Background(), "s1")
	if err != nil {
		t.Fatalf("list usage: %v", err)
	}
	if len(usage) != 1 {
		t.Fatalf("expected 1 usage record, got %d", len(usage))
	}
}
