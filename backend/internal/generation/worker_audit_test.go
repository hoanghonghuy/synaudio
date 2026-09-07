package generation

import (
	"context"
	"errors"
	"testing"
)

func TestProcessJobEmitsAuditBeforeSuccessTransition(t *testing.T) {
	store := newQueueFakeStore()
	svc := NewService(store)
	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")
	job, _ := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)

	var audited JobAuditEvent
	worker := NewWorker(svc, "w1", func(context.Context, GenerationJob) error { return nil }, WithJobAudit(
		func(_ context.Context, event JobAuditEvent) error {
			audited = event
			return nil
		},
	))
	if err := worker.ProcessOne(context.Background()); err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}
	if audited.Job.ID != job.ID || audited.Outcome != "SUCCEEDED" || audited.AttemptID == "" {
		t.Fatalf("unexpected worker audit event: %#v", audited)
	}
	for _, candidate := range store.jobs[run.ID] {
		if candidate.ID == job.ID && candidate.Status != "SUCCEEDED" {
			t.Fatalf("expected terminal success after audit, got %q", candidate.Status)
		}
	}
}

func TestAuditFailureLeavesJobRunningForReclaim(t *testing.T) {
	store := newQueueFakeStore()
	svc := NewService(store)
	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")
	job, _ := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)

	worker := NewWorker(svc, "w1", func(context.Context, GenerationJob) error { return nil }, WithJobAudit(
		func(context.Context, JobAuditEvent) error { return errors.New("audit unavailable") },
	))
	if err := worker.ProcessOne(context.Background()); err == nil {
		t.Fatal("expected audit persistence failure")
	}
	for _, candidate := range store.jobs[run.ID] {
		if candidate.ID == job.ID && candidate.Status != "RUNNING" {
			t.Fatalf("expected RUNNING lease for stale reclaim, got %q", candidate.Status)
		}
	}
}

func TestFailedJobAuditCarriesFailureClassification(t *testing.T) {
	store := newQueueFakeStore()
	svc := NewService(store)
	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")
	_, _ = svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)

	var audited JobAuditEvent
	worker := NewWorker(svc, "w1", func(context.Context, GenerationJob) error {
		return &ClassifiedError{Class: "PERMANENT", Code: "BAD_INPUT", Err: errors.New("bad")}
	}, WithJobAudit(func(_ context.Context, event JobAuditEvent) error {
		audited = event
		return nil
	}))
	if err := worker.ProcessOne(context.Background()); err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}
	if audited.Outcome != "FAILED" || audited.ErrorClass != "PERMANENT" || audited.ErrorCode != "BAD_INPUT" {
		t.Fatalf("unexpected failure audit event: %#v", audited)
	}
}

func TestProcessOneShutdownCancellationLeavesLeaseRunningWithoutFailureAudit(t *testing.T) {
	store := newQueueFakeStore()
	svc := NewService(store)
	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")
	job, _ := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)

	ctx, cancel := context.WithCancel(context.Background())
	audited := false
	worker := NewWorker(svc, "w1", func(context.Context, GenerationJob) error {
		cancel()
		return context.Canceled
	}, WithJobAudit(func(_ context.Context, _ JobAuditEvent) error {
		audited = true
		return nil
	}))

	err := worker.ProcessOne(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if audited {
		t.Fatal("shutdown cancellation must not emit a FAILED audit event")
	}
	for _, candidate := range store.jobs[run.ID] {
		if candidate.ID != job.ID {
			continue
		}
		if candidate.Status != "RUNNING" {
			t.Fatalf("expected RUNNING lease for expiry/reclaim, got %q", candidate.Status)
		}
		if candidate.LockedBy != "w1" {
			t.Fatalf("expected lease to remain owned by w1, got %q", candidate.LockedBy)
		}
	}
}

func TestProcessOneUnownedCancellationStillUsesFailurePolicy(t *testing.T) {
	store := newQueueFakeStore()
	svc := NewService(store)
	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")
	job, _ := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)

	var audited JobAuditEvent
	worker := NewWorker(svc, "w1", func(context.Context, GenerationJob) error {
		return context.Canceled
	}, WithJobAudit(func(_ context.Context, event JobAuditEvent) error {
		audited = event
		return nil
	}))

	if err := worker.ProcessOne(context.Background()); err != nil {
		t.Fatalf("ProcessOne: %v", err)
	}
	if audited.Outcome != "FAILED" || audited.ErrorClass != "PERMANENT" || audited.ErrorCode != "UNKNOWN" {
		t.Fatalf("expected normal failure policy for unowned cancellation, got %#v", audited)
	}
	for _, candidate := range store.jobs[run.ID] {
		if candidate.ID == job.ID && candidate.Status != "FAILED" {
			t.Fatalf("expected FAILED terminal state, got %q", candidate.Status)
		}
	}
}
