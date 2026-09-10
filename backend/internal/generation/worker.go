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

// ErrorClass exposes the bounded failure class to cross-cutting safe logging
// without coupling the logging package back to the generation domain.
func (e *ClassifiedError) ErrorClass() string { return e.Class }

// ErrorCode exposes the bounded failure code to cross-cutting safe logging.
func (e *ClassifiedError) ErrorCode() string { return e.Code }

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
	Duration   time.Duration
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
	startedAt := time.Now()
	procErr := w.process(ctx, job)
	duration := time.Since(startedAt)
	if procErr == nil {
		if w.audit != nil {
			if err := w.audit(ctx, JobAuditEvent{Job: job, AttemptID: attempt.ID, Outcome: "SUCCEEDED", Duration: duration}); err != nil {
				return err
			}
		}
		if err := w.svc.CompleteJobAttempt(ctx, attempt.ID, "SUCCEEDED", "", ""); err != nil {
			return err
		}
		return w.svc.MarkJobSucceeded(ctx, job.ID)
	}

	class, code := ClassifyError(procErr)
	if w.audit != nil {
		if err := w.audit(ctx, JobAuditEvent{Job: job, AttemptID: attempt.ID, Outcome: "FAILED", ErrorClass: class, ErrorCode: code, Duration: duration}); err != nil {
			return err
		}
	}
	if err := w.svc.CompleteJobAttempt(ctx, attempt.ID, "FAILED", class, code); err != nil {
		return err
	}
	return w.svc.MarkJobFailed(ctx, job.ID, class, code)
}
