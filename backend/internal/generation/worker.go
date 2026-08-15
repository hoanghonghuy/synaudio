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

// Worker claims and processes jobs from the queue.
type Worker struct {
	svc      *Service
	workerID string
	process  JobProcessor
}

func NewWorker(svc *Service, workerID string, process JobProcessor) *Worker {
	return &Worker{svc: svc, workerID: workerID, process: process}
}

// ProcessOne claims a single job, runs it, and records the outcome.
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
		_, _ = w.svc.CompleteJob(ctx, job.ID, "SUCCEEDED", "", "")
		_, _ = w.svc.store.UpdateJobAttemptStatus(ctx, attempt.ID, "SUCCEEDED", "", "")
		return nil
	}

	class, code := ClassifyError(runErr)
	_, _ = w.svc.store.UpdateJobAttemptStatus(ctx, attempt.ID, "FAILED", class, code)

	if shouldRetry(class, job.AttemptCount, job.MaxAttempts) {
		// Requeue: reset to PENDING so another worker can pick it up later.
		_, _ = w.svc.store.UpdateJobStatus(ctx, job.ID, "PENDING", class, code)
		return nil
	}

	_, _ = w.svc.CompleteJob(ctx, job.ID, "FAILED", class, code)
	return nil
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
