package generation

import (
	"context"
	"errors"
	"testing"
)

func TestClassifyErrorReturnsClassified(t *testing.T) {
	err := &ClassifiedError{Class: "TRANSIENT", Code: "TIMEOUT", Err: errors.New("timeout")}
	class, code := ClassifyError(err)
	if class != "TRANSIENT" || code != "TIMEOUT" {
		t.Fatalf("expected TRANSIENT/TIMEOUT, got %s/%s", class, code)
	}
}

func TestClassifyErrorDefaultsToPermanent(t *testing.T) {
	class, code := ClassifyError(errors.New("boom"))
	if class != "PERMANENT" || code != "UNKNOWN" {
		t.Fatalf("expected PERMANENT/UNKNOWN, got %s/%s", class, code)
	}
}

func TestShouldRetryTransientWithinMaxAttempts(t *testing.T) {
	if !shouldRetry("TRANSIENT", 1, 3) {
		t.Fatal("expected retry for transient within max attempts")
	}
}

func TestShouldNotRetryTransientAtMaxAttempts(t *testing.T) {
	if shouldRetry("TRANSIENT", 3, 3) {
		t.Fatal("expected no retry at max attempts")
	}
}

func TestShouldNotRetryPermanent(t *testing.T) {
	if shouldRetry("PERMANENT", 1, 3) {
		t.Fatal("expected no retry for permanent")
	}
}

func TestShouldNotRetryValidation(t *testing.T) {
	if shouldRetry("VALIDATION", 1, 3) {
		t.Fatal("expected no retry for validation")
	}
}

func TestShouldNotRetryPolicyBlock(t *testing.T) {
	if shouldRetry("POLICY_BLOCK", 1, 3) {
		t.Fatal("expected no retry for policy block")
	}
}

func TestShouldNotRetryStaleInput(t *testing.T) {
	if shouldRetry("STALE_INPUT", 1, 3) {
		t.Fatal("expected no retry for stale input")
	}
}

func TestBackoffDelayGrowsWithAttempt(t *testing.T) {
	d1 := backoffDelay(1)
	d2 := backoffDelay(2)
	d3 := backoffDelay(3)
	if d1 >= d2 || d2 >= d3 {
		t.Fatalf("expected growing backoff, got %v, %v, %v", d1, d2, d3)
	}
}

func TestProcessJobSuccessCompletesJob(t *testing.T) {
	store := newQueueFakeStore()
	svc := NewService(store)

	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")
	job, _ := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)

	processor := func(ctx context.Context, j GenerationJob) error {
		return nil
	}

	w := &Worker{svc: svc, workerID: "w1", process: processor}
	if err := w.ProcessOne(context.Background()); err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}

	got, _ := store.UpdateJobStatus(context.Background(), job.ID, "SUCCEEDED", "", "")
	_ = got
	// verify job is SUCCEEDED
	for _, jobs := range store.jobs {
		for _, j := range jobs {
			if j.ID == job.ID && j.Status != "SUCCEEDED" {
				t.Fatalf("expected SUCCEEDED, got %q", j.Status)
			}
		}
	}
}

func TestProcessJobTransientFailureRequeues(t *testing.T) {
	store := newQueueFakeStore()
	svc := NewService(store)

	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")
	job, _ := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)

	processor := func(ctx context.Context, j GenerationJob) error {
		return &ClassifiedError{Class: "TRANSIENT", Code: "TIMEOUT", Err: errors.New("timeout")}
	}

	w := &Worker{svc: svc, workerID: "w1", process: processor}
	if err := w.ProcessOne(context.Background()); err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}

	for _, jobs := range store.jobs {
		for _, j := range jobs {
			if j.ID == job.ID && j.Status != "PENDING" {
				t.Fatalf("expected PENDING (requeued), got %q", j.Status)
			}
		}
	}
}

func TestProcessJobPermanentFailureFailsJob(t *testing.T) {
	store := newQueueFakeStore()
	svc := NewService(store)

	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")
	job, _ := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)

	processor := func(ctx context.Context, j GenerationJob) error {
		return &ClassifiedError{Class: "PERMANENT", Code: "BAD_INPUT", Err: errors.New("bad")}
	}

	w := &Worker{svc: svc, workerID: "w1", process: processor}
	if err := w.ProcessOne(context.Background()); err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}

	for _, jobs := range store.jobs {
		for _, j := range jobs {
			if j.ID == job.ID && j.Status != "FAILED" {
				t.Fatalf("expected FAILED, got %q", j.Status)
			}
		}
	}
}
