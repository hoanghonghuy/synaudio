package pgstore

import (
	"context"

	"github.com/synaudio/synaudio/backend/internal/generation"
)

// ReclaimExpiredJobs reclaims only jobs whose persisted lease has expired.
// Lease duration/expiry is owned by the database claim/reclaim contract; the
// worker runtime does not supply an independent staleness threshold.
func (s *GenerationStore) ReclaimExpiredJobs(ctx context.Context) ([]generation.GenerationJob, error) {
	rows, err := s.q.ReclaimStaleJobs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]generation.GenerationJob, 0, len(rows))
	for _, r := range rows {
		out = append(out, toGenerationJob(r))
	}
	return out, nil
}
