package generation

import (
	"context"
)

// ReclaimStaleJobs resets RUNNING jobs whose lease has expired back to PENDING.
func (s *Service) ReclaimStaleJobs(ctx context.Context, olderThan string) ([]GenerationJob, error) {
	return s.store.ReclaimStaleJobs(ctx, olderThan)
}

// CancelJob marks a job as CANCELLED, stopping future attempts.
func (s *Service) CancelJob(ctx context.Context, jobID string) (GenerationJob, error) {
	return s.store.CancelJob(ctx, jobID)
}
