package generation

import (
	"context"
)

// ReclaimExpiredJobs resets RUNNING jobs whose durable lease has expired back
// to PENDING. Callers do not select a staleness threshold; persistence owns the
// lease-expiry authority.
func (s *Service) ReclaimExpiredJobs(ctx context.Context) ([]GenerationJob, error) {
	if reclaimer, ok := s.store.(interface {
		ReclaimExpiredJobs(context.Context) ([]GenerationJob, error)
	}); ok {
		return reclaimer.ReclaimExpiredJobs(ctx)
	}

	// Compatibility path for in-memory/test stores that still satisfy the older
	// Store interface. The empty value is intentionally non-authoritative; no
	// caller-selected duration crosses the service boundary.
	return s.store.ReclaimStaleJobs(ctx, "")
}

// ReclaimStaleJobs is retained temporarily for source compatibility. The
// threshold argument is intentionally ignored; durable lock expiry is the only
// reclaim authority. New runtime code must use ReclaimExpiredJobs.
func (s *Service) ReclaimStaleJobs(ctx context.Context, _ string) ([]GenerationJob, error) {
	return s.ReclaimExpiredJobs(ctx)
}

// CancelJob marks a job as CANCELLED, stopping future attempts.
func (s *Service) CancelJob(ctx context.Context, jobID string) (GenerationJob, error) {
	return s.store.CancelJob(ctx, jobID)
}
