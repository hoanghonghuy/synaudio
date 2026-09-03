package generation

import (
	"context"
	"errors"
	"time"
)

// ClassifiedError carries a failure class and code for retry decisions.
type ClassifiedError struct {
	Class string
	Code  string
	Err   error
}

func (e *ClassifiedError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}

func (e *ClassifiedError) Unwrap() error { return e.Err }

// ClassifyError maps a raw error to a failure class.
func ClassifyError(err error) (string, string) {
	var ce *ClassifiedError
	if errors.As(err, &ce) {
		return ce.Class, ce.Code
	}
	return "PERMANENT", "UNKNOWN"
}

// JobProcessor executes a single claimed job and returns an error on failure.
type JobProcessor func(ctx context.Context, job GenerationJob) error

// JobAuditEvent is a provider-agnostic worker lifecycle event. The generation
// package owns only execution semantics; the runtime decides how to persist it.
type JobAuditEvent struct {
	Job        GenerationJob
	AttemptID  string
	Outcome    string
	ErrorClass string
	ErrorCode  string
}

type JobAuditFunc func(ctx context.Context, event JobAuditEvent) error

type WorkerOption func(*Worker)

func WithJobAudit(record JobAuditFunc) WorkerOption {
	return func(w *Worker) {
		w.audit = record
	}
}

// Worker claims and processes jobs from the queue.
type Worker struct {
	svc      *Service
	workerID string
	process  JobProcessor
	audit    JobAuditFunc
}

func NewWorker(svc *Service, workerID string, process JobProcessor, opts ...WorkerOption) *Worker {
	w := &Worker{svc: svc, workerID: workerID, process: process}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// ProcessOne claims a single job, runs it, audits the outcome, and records the
// queue transition. Terminal/requeue transitions only happen after the audit
// callback succeeds, so a critical worker outcome cannot silently become
// unauditable. A failed audit leaves the lease RUNNING for normal stale reclaim.
func (w *Worker) ProcessOne(ctx context.Context) error {
	job, err := w.svc.ClaimNextJob(ctx, w.workerID)
	if err != nil {
		return err
	}

	attempt, err := w.svc.CreateJobAttempt(ctx, job.ID, "", "")
	if err != nil {
		return err
	}

	runErr := w.process(ctx, job)

	if runErr == nil {
		if err := w.recordAudit(ctx, JobAuditEvent{
			Job:       job,
			AttemptID: attempt.ID,
			Outcome:   "SUCCEEDED",
		}); err != nil {
			return err
		}
		if _, err := w.svc.CompleteJob(ctx, job.ID, "SUCCEEDED", "", ""); err != nil {
			return err
		}
		if _, err := w.svc.store.UpdateJobAttemptStatus(ctx, attempt.ID, "SUCCEEDED", "", ""); err != nil {
			return err
		}
		return nil
	}

	class, code := ClassifyError(runErr)
	if err := w.recordAudit(ctx, JobAuditEvent{
		Job:        job,
		AttemptID:  attempt.ID,
		Outcome:    "FAILED",
		ErrorClass: class,
		ErrorCode:  code,
	}); err != nil {
		return err
	}
	if _, err := w.svc.store.UpdateJobAttemptStatus(ctx, attempt.ID, "FAILED", class, code); err != nil {
		return err
	}

	if shouldRetry(class, job.AttemptCount, job.MaxAttempts) {
		if _, err := w.svc.store.UpdateJobStatus(ctx, job.ID, "PENDING", class, code); err != nil {
			return err
		}
		return nil
	}

	_, err = w.svc.CompleteJob(ctx, job.ID, "FAILED", class, code)
	return err
}

func (w *Worker) recordAudit(ctx context.Context, event JobAuditEvent) error {
	if w.audit == nil {
		return nil
	}
	return w.audit(ctx, event)
}

// shouldRetry reports whether a job should be retried given its failure class.
func shouldRetry(class string, attemptCount, maxAttempts int) bool {
	if class != "TRANSIENT" {
		return false
	}
	return attemptCount < maxAttempts
}

// backoffDelay returns the delay before retrying a given attempt (1-indexed).
func backoffDelay(attempt int) time.Duration {
	base := time.Duration(attempt) * 2 * time.Second
	return base
}
