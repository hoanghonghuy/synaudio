package generation

import (
	"context"
	"errors"
	"testing"
)

type queueFakeStore struct {
	*fakeStore
	claimed []string
}

func newQueueFakeStore() *queueFakeStore {
	return &queueFakeStore{
		fakeStore: newFakeStore(),
		claimed:   []string{},
	}
}

func (s *queueFakeStore) ClaimNextJob(ctx context.Context, workerID string) (GenerationJob, error) {
	for runID, jobs := range s.jobs {
		for i, j := range jobs {
			if j.Status == "PENDING" {
				j.Status = "RUNNING"
				j.LockedBy = workerID
				s.jobs[runID][i] = j
				s.claimed = append(s.claimed, j.ID)
				return j, nil
			}
		}
	}
	return GenerationJob{}, ErrNoRunnableJob
}

func (s *queueFakeStore) UpdateJobStatus(ctx context.Context, jobID, status, errorClass, errorCode string) (GenerationJob, error) {
	for runID, jobs := range s.jobs {
		for i, j := range jobs {
			if j.ID == jobID {
				j.Status = status
				j.LastErrorClass = errorClass
				j.LastErrorCode = errorCode
				s.jobs[runID][i] = j
				return j, nil
			}
		}
	}
	return GenerationJob{}, ErrGenerationJobNotFound
}

func (s *queueFakeStore) UpdateJobAttemptStatus(ctx context.Context, attemptID, status, errorClass, errorCode string) (JobAttempt, error) {
	return JobAttempt{ID: attemptID, Status: status, ErrorClass: errorClass, ErrorCode: errorCode}, nil
}

func TestClaimNextJobReturnsPendingJob(t *testing.T) {
	store := newQueueFakeStore()
	svc := NewService(store)

	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")
	job, _ := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)

	claimed, err := svc.ClaimNextJob(context.Background(), "worker-1")
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claimed.ID != job.ID {
		t.Fatalf("expected job %q, got %q", job.ID, claimed.ID)
	}
	if claimed.Status != "RUNNING" {
		t.Fatalf("expected RUNNING, got %q", claimed.Status)
	}
}

func TestClaimNextJobReturnsNoRunnableJob(t *testing.T) {
	store := newQueueFakeStore()
	svc := NewService(store)

	if _, err := svc.ClaimNextJob(context.Background(), "worker-1"); !errors.Is(err, ErrNoRunnableJob) {
		t.Fatalf("expected ErrNoRunnableJob, got %v", err)
	}
}

func TestCompleteJobMarksSucceeded(t *testing.T) {
	store := newQueueFakeStore()
	svc := NewService(store)

	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")
	job, _ := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)

	completed, err := svc.CompleteJob(context.Background(), job.ID, "SUCCEEDED", "", "")
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if completed.Status != "SUCCEEDED" {
		t.Fatalf("expected SUCCEEDED, got %q", completed.Status)
	}
}

func TestFailJobMarksFailedWithError(t *testing.T) {
	store := newQueueFakeStore()
	svc := NewService(store)

	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")
	job, _ := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)

	failed, err := svc.CompleteJob(context.Background(), job.ID, "FAILED", "TRANSIENT", "TIMEOUT")
	if err != nil {
		t.Fatalf("fail: %v", err)
	}
	if failed.Status != "FAILED" {
		t.Fatalf("expected FAILED, got %q", failed.Status)
	}
	if failed.LastErrorClass != "TRANSIENT" {
		t.Fatalf("expected TRANSIENT, got %q", failed.LastErrorClass)
	}
	if failed.LastErrorCode != "TIMEOUT" {
		t.Fatalf("expected TIMEOUT, got %q", failed.LastErrorCode)
	}
}
