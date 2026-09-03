package generation

import (
	"context"
	"errors"
	"testing"
)

func (s *fakeStore) EstablishWriterBatch(_ context.Context, run GenerationRun, jobs []WriterBatchJob) (GenerationRun, error) {
	stagedInputs := make(map[string]WriterJobInput, len(jobs))
	stagedJobs := make([]GenerationJob, 0, len(jobs))
	for _, item := range jobs {
		input, ok := s.currentWriterPlans[item.ChapterID]
		if !ok {
			input = WriterJobInput{
				ChapterID:          item.ChapterID,
				PlanRevisionID:     "plan-" + item.ChapterID,
				Plan:               map[string]any{"chapter_id": item.ChapterID},
				BaseCanonVersionID: "canon-" + item.ChapterID,
			}
		}
		input.JobID = item.Job.ID
		input.ChapterID = item.ChapterID
		input.Plan = clonePlan(input.Plan)
		stagedInputs[item.Job.ID] = input
		stagedJobs = append(stagedJobs, item.Job)
	}

	s.runs[run.StoryID] = append(s.runs[run.StoryID], run)
	s.jobs[run.ID] = append(s.jobs[run.ID], stagedJobs...)
	for jobID, input := range stagedInputs {
		s.writerInputs[jobID] = input
	}
	return run, nil
}

type controlledBatchStore struct {
	*fakeStore
	plans  map[string]WriterJobInput
	failAt int
}

func newControlledBatchStore(plans map[string]WriterJobInput, failAt int) *controlledBatchStore {
	return &controlledBatchStore{fakeStore: newFakeStore(), plans: plans, failAt: failAt}
}

func (s *controlledBatchStore) EstablishWriterBatch(_ context.Context, run GenerationRun, jobs []WriterBatchJob) (GenerationRun, error) {
	stagedInputs := make(map[string]WriterJobInput, len(jobs))
	stagedJobs := make([]GenerationJob, 0, len(jobs))
	for i, item := range jobs {
		if s.failAt >= 0 && i == s.failAt {
			return GenerationRun{}, errors.New("injected batch persistence failure")
		}
		input, ok := s.plans[item.ChapterID]
		if !ok || input.PlanRevisionID == "" || input.Plan == nil {
			return GenerationRun{}, ErrWriterPlanNotBound
		}
		input.JobID = item.Job.ID
		input.ChapterID = item.ChapterID
		input.Plan = clonePlan(input.Plan)
		stagedInputs[item.Job.ID] = input
		stagedJobs = append(stagedJobs, item.Job)
	}

	s.runs[run.StoryID] = append(s.runs[run.StoryID], run)
	s.jobs[run.ID] = append(s.jobs[run.ID], stagedJobs...)
	for jobID, input := range stagedInputs {
		s.writerInputs[jobID] = input
	}
	return run, nil
}

func clonePlan(plan map[string]any) map[string]any {
	if plan == nil {
		return nil
	}
	out := make(map[string]any, len(plan))
	for key, value := range plan {
		out[key] = value
	}
	return out
}

func writerPlan(chapterID, revisionID string) WriterJobInput {
	return WriterJobInput{
		ChapterID:          chapterID,
		PlanRevisionID:     revisionID,
		Plan:               map[string]any{"chapter_id": chapterID, "revision": revisionID},
		BaseCanonVersionID: "canon-1",
	}
}

func TestStartBatchGenerationMissingPlanLaterLeavesNoExecutableWork(t *testing.T) {
	store := newControlledBatchStore(map[string]WriterJobInput{
		"ch1": writerPlan("ch1", "plan-1"),
	}, -1)
	svc := NewService(store)

	if _, err := svc.StartBatchGeneration(context.Background(), "s1", []string{"ch1", "ch2"}, "u1"); !errors.Is(err, ErrWriterPlanNotBound) {
		t.Fatalf("expected ErrWriterPlanNotBound, got %v", err)
	}
	if len(store.runs["s1"]) != 0 {
		t.Fatalf("failed batch persisted %d generation runs", len(store.runs["s1"]))
	}
	if _, err := store.ClaimNextJob(context.Background(), "worker-1"); !errors.Is(err, ErrNoRunnableJob) {
		t.Fatalf("failed batch left claimable work: %v", err)
	}
}

func TestStartBatchGenerationPersistenceFailureMidBatchLeavesNoExecutableWork(t *testing.T) {
	store := newControlledBatchStore(map[string]WriterJobInput{
		"ch1": writerPlan("ch1", "plan-1"),
		"ch2": writerPlan("ch2", "plan-2"),
		"ch3": writerPlan("ch3", "plan-3"),
	}, 1)
	svc := NewService(store)

	if _, err := svc.StartBatchGeneration(context.Background(), "s1", []string{"ch1", "ch2", "ch3"}, "u1"); err == nil {
		t.Fatal("expected injected persistence failure")
	}
	if len(store.runs["s1"]) != 0 {
		t.Fatalf("failed batch persisted %d generation runs", len(store.runs["s1"]))
	}
	if _, err := store.ClaimNextJob(context.Background(), "worker-1"); !errors.Is(err, ErrNoRunnableJob) {
		t.Fatalf("failed batch left claimable work: %v", err)
	}
}

func TestStartBatchGenerationPreservesExactPlanPerJob(t *testing.T) {
	store := newControlledBatchStore(map[string]WriterJobInput{
		"ch1": writerPlan("ch1", "plan-17"),
		"ch2": writerPlan("ch2", "plan-29"),
	}, -1)
	svc := NewService(store)

	run, err := svc.StartBatchGeneration(context.Background(), "s1", []string{"ch1", "ch2"}, "u1")
	if err != nil {
		t.Fatalf("start batch: %v", err)
	}
	jobs := store.jobs[run.ID]
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
	if got := store.writerInputs[jobs[0].ID].PlanRevisionID; got != "plan-17" {
		t.Fatalf("first job froze %q, want plan-17", got)
	}
	if got := store.writerInputs[jobs[1].ID].PlanRevisionID; got != "plan-29" {
		t.Fatalf("second job froze %q, want plan-29", got)
	}
}
