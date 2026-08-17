package generation

import (
	"context"
	"errors"
)

var (
	ErrNoRunnableJob       = errors.New("no runnable job")
	ErrGenerationJobNotFound = errors.New("generation job not found")
	ErrJobAttemptNotFound  = errors.New("job attempt not found")
)

// ClaimNextJob claims the next PENDING job for a worker.
func (s *Service) ClaimNextJob(ctx context.Context, workerID string) (GenerationJob, error) {
	return s.store.ClaimNextJob(ctx, workerID)
}

// CompleteJob marks a job as SUCCEEDED or FAILED with optional error info.
func (s *Service) CompleteJob(ctx context.Context, jobID, status, errorClass, errorCode string) (GenerationJob, error) {
	if status == "" {
		status = "SUCCEEDED"
	}
	return s.store.UpdateJobStatus(ctx, jobID, status, errorClass, errorCode)
}
