package generation

import (
	"context"
	"errors"
	"testing"
)

func TestRetryGenerationJobRequeuesFailedTransientJob(t *testing.T) {
	store := newFakeStore()
	store.currentWriterPlans["c1"] = WriterJobInput{ChapterID: "c1", PlanRevisionID: "plan-1", Plan: map[string]any{"beat": "one"}}
	run, _ := store.CreateGenerationRun(context.Background(), GenerationRun{ID: "run-1", RunType: "CHAPTER_GENERATION", StoryID: "s1"})
	job := GenerationJob{
		ID:             "job-1",
		RunID:          run.ID,
		JobType:        "WRITER",
		Status:         "FAILED",
		AttemptCount:   1,
		MaxAttempts:    3,
		LastErrorClass: "TRANSIENT",
		LastErrorCode:  "PROVIDER_TIMEOUT",
	}
	store.jobs[run.ID] = []GenerationJob{job}
	store.writerInputs[job.ID] = store.currentWriterPlans["c1"]

	svc := NewService(store)
	view, err := svc.RetryGenerationJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if view.Observation != "queued" || view.Retryable {
		t.Fatalf("expected queued after retry, got %#v", view)
	}
	if stored := store.jobs[run.ID][0]; stored.Status != "PENDING" || stored.AttemptCount != 1 {
		t.Fatalf("retry must not reset attempt_count: %#v", stored)
	}
}

func TestRetryGenerationJobRejectsExhaustedJob(t *testing.T) {
	store := newFakeStore()
	run, _ := store.CreateGenerationRun(context.Background(), GenerationRun{ID: "run-1", RunType: "CHAPTER_GENERATION", StoryID: "s1"})
	job := GenerationJob{
		ID:             "job-1",
		RunID:          run.ID,
		JobType:        "WRITER",
		Status:         "FAILED",
		AttemptCount:   3,
		MaxAttempts:    3,
		LastErrorClass: "RETRY_EXHAUSTED",
		LastErrorCode:  "MAX_ATTEMPTS_EXHAUSTED",
	}
	store.jobs[run.ID] = []GenerationJob{job}

	svc := NewService(store)
	if _, err := svc.RetryGenerationJob(context.Background(), job.ID); !errors.Is(err, ErrJobNotRetryable) {
		t.Fatalf("expected ErrJobNotRetryable, got %v", err)
	}
}

func TestRetryGenerationJobIsIdempotentForPending(t *testing.T) {
	store := newFakeStore()
	store.currentWriterPlans["c1"] = WriterJobInput{ChapterID: "c1", PlanRevisionID: "plan-1", Plan: map[string]any{}}
	run, _ := store.CreateGenerationRun(context.Background(), GenerationRun{ID: "run-1", RunType: "CHAPTER_GENERATION", StoryID: "s1"})
	job := GenerationJob{ID: "job-1", RunID: run.ID, JobType: "WRITER", Status: "PENDING", MaxAttempts: 3}
	store.jobs[run.ID] = []GenerationJob{job}
	store.writerInputs[job.ID] = store.currentWriterPlans["c1"]

	svc := NewService(store)
	first, err := svc.RetryGenerationJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("first retry: %v", err)
	}
	second, err := svc.RetryGenerationJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("second retry: %v", err)
	}
	if first.Observation != "queued" || second.Observation != "queued" {
		t.Fatalf("expected queued idempotent views: %#v %#v", first, second)
	}
}

func TestRetryGenerationJobDoesNotDoubleCommitWriterOutput(t *testing.T) {
	store := newFakeStore()
	store.currentWriterPlans["c1"] = WriterJobInput{
		ChapterID:      "c1",
		PlanRevisionID: "plan-v1",
		Plan:           map[string]any{"beat": "one"},
	}
	_, _ = store.CreateGenerationRun(context.Background(), GenerationRun{ID: "run-1", RunType: "CHAPTER_GENERATION", StoryID: "s1", ChapterID: "c1", RequestedBy: "user-1"})
	provider := &captureTextAI{text: "Generated chapter prose."}
	svc := NewService(store, WithTextAI(provider))
	job, err := svc.CreateGenerationJob(context.Background(), "run-1", "WRITER", 3)
	if err != nil {
		t.Fatalf("create writer job: %v", err)
	}

	first, err := svc.ExecuteWriterJob(context.Background(), job)
	if err != nil {
		t.Fatalf("first execution: %v", err)
	}
	_, _ = store.UpdateJobStatus(context.Background(), job.ID, "FAILED", "TRANSIENT", "PROVIDER_TIMEOUT")

	view, err := svc.RetryGenerationJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if view.Observation != "queued" {
		t.Fatalf("expected queued after retry, got %#v", view)
	}

	requeued, err := store.GetGenerationJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	second, err := svc.ExecuteWriterJob(context.Background(), requeued)
	if err != nil {
		t.Fatalf("retry execution: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("retry created duplicate revision: first=%q second=%q", first.ID, second.ID)
	}
	if provider.calls != 1 {
		t.Fatalf("expected provider once across retry, got %d", provider.calls)
	}
}
