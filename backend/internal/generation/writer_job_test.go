package generation

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type captureTextAI struct {
	text    string
	err     error
	calls   int
	prompts []string
}

func (p *captureTextAI) GenerateText(_ context.Context, in TextAIInput) (TextAIOutput, error) {
	p.calls++
	p.prompts = append(p.prompts, in.Prompt)
	if p.err != nil {
		return TextAIOutput{}, p.err
	}
	return TextAIOutput{Text: p.text, Provider: "test", Model: "writer-test"}, nil
}

func TestWriterJobUsesFrozenPlanAndPersistsDurableProvenance(t *testing.T) {
	store := newFakeStore()
	store.currentWriterPlans["c1"] = WriterJobInput{
		ChapterID:          "c1",
		PlanRevisionID:     "plan-v1",
		Plan:               map[string]any{"beat": "frozen opening"},
		BaseCanonVersionID: "canon-plan-v1",
	}
	_, _ = store.CreateGenerationRun(context.Background(), GenerationRun{
		ID:                 "run-1",
		RunType:            "CHAPTER_GENERATION",
		StoryID:            "story-1",
		ChapterID:          "c1",
		Status:             "PENDING",
		WorkflowVersion:    "writer-workflow-v1",
		BaseCanonVersionID: "canon-run-1",
		RequestedBy:        "user-1",
	})

	provider := &captureTextAI{text: "Generated chapter prose."}
	svc := NewService(store, WithTextAI(provider))
	job, err := svc.CreateGenerationJob(context.Background(), "run-1", "WRITER", 3)
	if err != nil {
		t.Fatalf("create writer job: %v", err)
	}

	// Advance mutable chapter-head state after enqueue. Execution must remain on v1.
	store.currentWriterPlans["c1"] = WriterJobInput{
		ChapterID:          "c1",
		PlanRevisionID:     "plan-v2",
		Plan:               map[string]any{"beat": "new mutable opening"},
		BaseCanonVersionID: "canon-plan-v2",
	}

	revision, err := svc.ExecuteWriterJob(context.Background(), job)
	if err != nil {
		t.Fatalf("execute writer job: %v", err)
	}
	if revision.PlanRevisionID != "plan-v1" {
		t.Fatalf("expected frozen plan-v1, got %q", revision.PlanRevisionID)
	}
	if revision.GenerationRunID != "run-1" {
		t.Fatalf("expected run provenance run-1, got %q", revision.GenerationRunID)
	}
	if revision.BaseCanonVersionID != "canon-run-1" {
		t.Fatalf("expected frozen run canon, got %q", revision.BaseCanonVersionID)
	}
	if revision.CreatedBy != "user-1" {
		t.Fatalf("expected requester provenance user-1, got %q", revision.CreatedBy)
	}
	if provider.calls != 1 {
		t.Fatalf("expected one provider call, got %d", provider.calls)
	}
	if !strings.Contains(provider.prompts[0], "frozen opening") {
		t.Fatalf("writer prompt did not contain frozen plan: %s", provider.prompts[0])
	}
	if strings.Contains(provider.prompts[0], "new mutable opening") || strings.Contains(provider.prompts[0], "plan-v2") {
		t.Fatalf("writer prompt leaked later mutable plan: %s", provider.prompts[0])
	}

	storedJob := store.jobs["run-1"][0]
	if storedJob.OutputRef["content_revision_id"] != revision.ID {
		t.Fatalf("expected durable output ref %q, got %#v", revision.ID, storedJob.OutputRef)
	}
	if storedJob.OutputRef["plan_revision_id"] != "plan-v1" {
		t.Fatalf("expected output plan-v1, got %#v", storedJob.OutputRef)
	}
}

func TestWriterJobRetryReusesExistingDurableOutput(t *testing.T) {
	store := newFakeStore()
	store.currentWriterPlans["c1"] = WriterJobInput{
		ChapterID:      "c1",
		PlanRevisionID: "plan-v1",
		Plan:           map[string]any{"beat": "one"},
	}
	_, _ = store.CreateGenerationRun(context.Background(), GenerationRun{
		ID:        "run-1",
		RunType:   "CHAPTER_GENERATION",
		StoryID:   "story-1",
		ChapterID: "c1",
		Status:    "PENDING",
	})

	provider := &captureTextAI{text: "Generated once."}
	svc := NewService(store, WithTextAI(provider))
	job, err := svc.CreateGenerationJob(context.Background(), "run-1", "WRITER", 3)
	if err != nil {
		t.Fatalf("create writer job: %v", err)
	}

	first, err := svc.ExecuteWriterJob(context.Background(), job)
	if err != nil {
		t.Fatalf("first execution: %v", err)
	}
	second, err := svc.ExecuteWriterJob(context.Background(), job)
	if err != nil {
		t.Fatalf("retry execution: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected retry to reuse revision %q, got %q", first.ID, second.ID)
	}
	if provider.calls != 1 {
		t.Fatalf("expected provider to run once across retry, got %d", provider.calls)
	}
	if len(store.revisions["c1"]) != 1 {
		t.Fatalf("expected one durable content revision, got %d", len(store.revisions["c1"]))
	}
}

func TestWriterJobMissingFrozenInputIsPermanentFailure(t *testing.T) {
	store := newFakeStore()
	_, _ = store.CreateGenerationRun(context.Background(), GenerationRun{
		ID:        "run-1",
		RunType:   "CHAPTER_GENERATION",
		StoryID:   "story-1",
		ChapterID: "c1",
		Status:    "PENDING",
	})
	job, _ := store.CreateGenerationJob(context.Background(), GenerationJob{
		ID:          "job-unbound",
		RunID:       "run-1",
		JobType:     "WRITER",
		Status:      "PENDING",
		MaxAttempts: 3,
	})

	svc := NewService(store, WithTextAI(&captureTextAI{text: "must not run"}))
	_, err := svc.ExecuteWriterJob(context.Background(), job)
	if err == nil {
		t.Fatal("expected missing frozen writer input to fail")
	}
	class, code := ClassifyError(err)
	if class != "PERMANENT" || code != "WRITER_INPUT_MISSING" {
		t.Fatalf("expected PERMANENT/WRITER_INPUT_MISSING, got %s/%s (%v)", class, code, err)
	}
	if len(store.revisions["c1"]) != 0 {
		t.Fatal("missing input must not create content")
	}
}

func TestWriterProviderFailureIsRetryableAndCreatesNoOutput(t *testing.T) {
	store := newFakeStore()
	store.currentWriterPlans["c1"] = WriterJobInput{
		ChapterID:      "c1",
		PlanRevisionID: "plan-v1",
		Plan:           map[string]any{"beat": "one"},
	}
	_, _ = store.CreateGenerationRun(context.Background(), GenerationRun{
		ID:        "run-1",
		RunType:   "CHAPTER_GENERATION",
		StoryID:   "story-1",
		ChapterID: "c1",
		Status:    "PENDING",
	})

	provider := &captureTextAI{err: errors.New("temporary provider failure")}
	svc := NewService(store, WithTextAI(provider))
	job, err := svc.CreateGenerationJob(context.Background(), "run-1", "WRITER", 3)
	if err != nil {
		t.Fatalf("create writer job: %v", err)
	}

	_, err = svc.ExecuteWriterJob(context.Background(), job)
	if err == nil {
		t.Fatal("expected provider failure")
	}
	class, code := ClassifyError(err)
	if class != "TRANSIENT" || code != "WRITER_PROVIDER_FAILED" {
		t.Fatalf("expected TRANSIENT/WRITER_PROVIDER_FAILED, got %s/%s (%v)", class, code, err)
	}
	if len(store.revisions["c1"]) != 0 {
		t.Fatal("provider failure must not create content")
	}
	if store.jobs["run-1"][0].OutputRef != nil {
		t.Fatal("provider failure must not record writer output")
	}
}
