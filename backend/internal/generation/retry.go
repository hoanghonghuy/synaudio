package generation

import (
	"context"
	"errors"
)

var (
	ErrJobNotRetryable = errors.New("generation job is not retryable")
)

// GetGenerationJob returns one durable job by ID.
func (s *Service) GetGenerationJob(ctx context.Context, jobID string) (GenerationJob, error) {
	return s.store.GetGenerationJob(ctx, jobID)
}

// GetLatestChapterWriterJob returns the newest WRITER job bound to one chapter.
func (s *Service) GetLatestChapterWriterJob(ctx context.Context, chapterID string) (GenerationJob, error) {
	return s.store.GetLatestWriterJobForChapter(ctx, chapterID)
}

// ObserveChapterWriterJob returns the authoritative observation for the latest
// chapter WRITER job, or nil when no durable job exists.
func (s *Service) ObserveChapterWriterJob(ctx context.Context, chapterID string) (*JobView, error) {
	job, err := s.GetLatestChapterWriterJob(ctx, chapterID)
	if err != nil {
		if errors.Is(err, ErrGenerationJobNotFound) {
			return nil, nil
		}
		return nil, err
	}
	view := ObserveJob(job)
	return &view, nil
}

// RetryGenerationJob re-queues a failed WRITER job using the existing attempt
// budget. It is idempotent for jobs already pending or running and never resets
// attempt_count.
func (s *Service) RetryGenerationJob(ctx context.Context, jobID string) (JobView, error) {
	job, err := s.store.GetGenerationJob(ctx, jobID)
	if err != nil {
		return JobView{}, err
	}

	switch job.Status {
	case "PENDING", "RUNNING":
		return ObserveJob(job), nil
	case "SUCCEEDED":
		return JobView{}, ErrJobNotRetryable
	}

	if !adminRetryPermitted(job) {
		return JobView{}, ErrJobNotRetryable
	}

	requeued, err := s.store.RequeueGenerationJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, ErrGenerationJobNotFound) {
			current, getErr := s.store.GetGenerationJob(ctx, jobID)
			if getErr != nil {
				return JobView{}, getErr
			}
			if current.Status == "PENDING" || current.Status == "RUNNING" {
				return ObserveJob(current), nil
			}
			return JobView{}, ErrJobNotRetryable
		}
		return JobView{}, err
	}

	return ObserveJob(requeued), nil
}
