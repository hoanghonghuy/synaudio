package generation

import (
	"context"
	"testing"
)

type reclaimFakeStore struct {
	*queueFakeStore
	reclaimed []string
}

func newReclaimFakeStore() *reclaimFakeStore {
	return &reclaimFakeStore{
		queueFakeStore: newQueueFakeStore(),
		reclaimed:      []string{},
	}
}

func (s *reclaimFakeStore) ReclaimStaleJobs(ctx context.Context, olderThan string) ([]GenerationJob, error) {
	out := []GenerationJob{}
	for runID, jobs := range s.jobs {
		for i, j := range jobs {
			if j.Status == "RUNNING" {
				j.Status = "PENDING"
				j.LockedBy = ""
				s.jobs[runID][i] = j
				s.reclaimed = append(s.reclaimed, j.ID)
				out = append(out, j)
			}
		}
	}
	return out, nil
}

func (s *reclaimFakeStore) CancelJob(ctx context.Context, jobID string) (GenerationJob, error) {
	for runID, jobs := range s.jobs {
		for i, j := range jobs {
			if j.ID == jobID {
				j.Status = "CANCELLED"
				s.jobs[runID][i] = j
				return j, nil
			}
		}
	}
	return GenerationJob{}, ErrGenerationJobNotFound
}

func TestReclaimStaleJobsResetsRunningToPending(t *testing.T) {
	store := newReclaimFakeStore()
	svc := NewService(store)

	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")
	job, _ := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)

	// Simulate a claimed (RUNNING) job.
	_, _ = svc.ClaimNextJob(context.Background(), "worker-1")

	reclaimed, err := svc.ReclaimStaleJobs(context.Background(), "5 minutes")
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if len(reclaimed) != 1 {
		t.Fatalf("expected 1 reclaimed, got %d", len(reclaimed))
	}
	if reclaimed[0].ID != job.ID {
		t.Fatalf("expected job %q, got %q", job.ID, reclaimed[0].ID)
	}
	if reclaimed[0].Status != "PENDING" {
		t.Fatalf("expected PENDING, got %q", reclaimed[0].Status)
	}
}

func TestCancelJobMarksCancelled(t *testing.T) {
	store := newReclaimFakeStore()
	svc := NewService(store)

	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")
	job, _ := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)

	cancelled, err := svc.CancelJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if cancelled.Status != "CANCELLED" {
		t.Fatalf("expected CANCELLED, got %q", cancelled.Status)
	}
}
