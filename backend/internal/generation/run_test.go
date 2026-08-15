package generation

import (
	"context"
	"errors"
	"testing"
)

type runFakeStore struct {
	*fakeStore
	runs  map[string][]GenerationRun
	jobs  map[string][]GenerationJob
	nextAttempt map[string]int
}

func newRunFakeStore() *runFakeStore {
	return &runFakeStore{
		fakeStore:   newFakeStore(),
		runs:        map[string][]GenerationRun{},
		jobs:        map[string][]GenerationJob{},
		nextAttempt: map[string]int{},
	}
}

func (s *runFakeStore) CreateGenerationRun(ctx context.Context, r GenerationRun) (GenerationRun, error) {
	s.runs[r.StoryID] = append(s.runs[r.StoryID], r)
	return r, nil
}

func (s *runFakeStore) GetGenerationRun(ctx context.Context, runID string) (GenerationRun, error) {
	for _, rs := range s.runs {
		for _, r := range rs {
			if r.ID == runID {
				return r, nil
			}
		}
	}
	return GenerationRun{}, ErrGenerationRunNotFound
}

func (s *runFakeStore) CreateGenerationJob(ctx context.Context, j GenerationJob) (GenerationJob, error) {
	s.jobs[j.RunID] = append(s.jobs[j.RunID], j)
	return j, nil
}

func (s *runFakeStore) NextAttemptNo(ctx context.Context, jobID string) (int, error) {
	s.nextAttempt[jobID]++
	return s.nextAttempt[jobID], nil
}

func (s *runFakeStore) CreateJobAttempt(ctx context.Context, a JobAttempt) (JobAttempt, error) {
	return a, nil
}

func TestCreateGenerationRun(t *testing.T) {
	store := newRunFakeStore()
	svc := NewService(store)

	r, err := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	if r.Status != "PENDING" {
		t.Fatalf("expected PENDING, got %q", r.Status)
	}
	if r.RunType != "CHAPTER_GENERATION" {
		t.Fatalf("expected CHAPTER_GENERATION, got %q", r.RunType)
	}
}

func TestCreateGenerationRunRejectsEmptyType(t *testing.T) {
	store := newRunFakeStore()
	svc := NewService(store)

	if _, err := svc.CreateGenerationRun(context.Background(), "  ", "s1", "c1", "u1"); err == nil {
		t.Fatal("expected error for empty run type")
	}
}

func TestCreateGenerationJob(t *testing.T) {
	store := newRunFakeStore()
	svc := NewService(store)

	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")

	j, err := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if j.Status != "PENDING" {
		t.Fatalf("expected PENDING, got %q", j.Status)
	}
	if j.MaxAttempts != 3 {
		t.Fatalf("expected max_attempts 3, got %d", j.MaxAttempts)
	}
}

func TestCreateJobAttempt(t *testing.T) {
	store := newRunFakeStore()
	svc := NewService(store)

	run, _ := svc.CreateGenerationRun(context.Background(), "CHAPTER_GENERATION", "s1", "c1", "u1")
	job, _ := svc.CreateGenerationJob(context.Background(), run.ID, "WRITER", 3)

	a, err := svc.CreateJobAttempt(context.Background(), job.ID, "mock", "mock-model")
	if err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	if a.AttemptNo != 1 {
		t.Fatalf("expected attempt 1, got %d", a.AttemptNo)
	}
	if a.Status != "RUNNING" {
		t.Fatalf("expected RUNNING, got %q", a.Status)
	}
}

func TestGetGenerationRunReturnsNotFound(t *testing.T) {
	store := newRunFakeStore()
	svc := NewService(store)

	if _, err := svc.GetGenerationRun(context.Background(), "missing"); !errors.Is(err, ErrGenerationRunNotFound) {
		t.Fatalf("expected ErrGenerationRunNotFound, got %v", err)
	}
}
