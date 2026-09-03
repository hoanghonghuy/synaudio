package generation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrWriterPlanNotBound      = errors.New("writer chapter plan is not bound")
	ErrWriterJobInputNotFound  = errors.New("writer job input not found")
)

// WriterJobInput is the immutable input snapshot for one WRITER job. The exact
// Chapter Plan revision is frozen when the job is enqueued; execution must never
// reconstruct it from chapters.current_plan_revision_id.
type WriterJobInput struct {
	JobID              string
	ChapterID          string
	PlanRevisionID     string
	Plan               map[string]any
	BaseCanonVersionID string
}

// WriterStore is the persistence contract needed only by durable WRITER jobs.
// Keeping it separate from Store avoids coupling unrelated generation paths to
// the physical job-input representation.
type WriterStore interface {
	CreateWriterGenerationJob(ctx context.Context, j GenerationJob, chapterID string) (GenerationJob, error)
	GetWriterJobInput(ctx context.Context, jobID string) (WriterJobInput, error)
	GetWriterOutput(ctx context.Context, runID, planRevisionID string) (ContentRevision, error)
	UpdateWriterJobOutputRef(ctx context.Context, jobID string, outputRef map[string]any) (GenerationJob, error)
}

func (s *Service) writerStore() (WriterStore, error) {
	store, ok := s.store.(WriterStore)
	if !ok {
		return nil, errors.New("writer persistence not configured")
	}
	return store, nil
}

func (s *Service) createWriterGenerationJob(ctx context.Context, runID, chapterID string, maxAttempts int) (GenerationJob, error) {
	chapterID = strings.TrimSpace(chapterID)
	if chapterID == "" {
		return GenerationJob{}, ErrWriterPlanNotBound
	}
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	store, err := s.writerStore()
	if err != nil {
		return GenerationJob{}, err
	}

	job := GenerationJob{
		ID:          uuid.NewString(),
		RunID:       runID,
		JobType:     "WRITER",
		Status:      "PENDING",
		MaxAttempts: maxAttempts,
	}
	return store.CreateWriterGenerationJob(ctx, job, chapterID)
}

// ExecuteWriterJob performs the durable business work represented by a WRITER
// job. It returns success only after a content revision with exact frozen
// provenance exists and the job output_ref records that revision.
func (s *Service) ExecuteWriterJob(ctx context.Context, job GenerationJob) (ContentRevision, error) {
	if job.JobType != "WRITER" {
		return ContentRevision{}, writerPermanent("WRITER_JOB_TYPE_INVALID", fmt.Errorf("expected WRITER job, got %q", job.JobType))
	}

	store, err := s.writerStore()
	if err != nil {
		return ContentRevision{}, writerPermanent("WRITER_STORE_NOT_CONFIGURED", err)
	}

	input, err := store.GetWriterJobInput(ctx, job.ID)
	if err != nil {
		if errors.Is(err, ErrWriterJobInputNotFound) {
			return ContentRevision{}, writerPermanent("WRITER_INPUT_MISSING", err)
		}
		return ContentRevision{}, writerTransient("WRITER_INPUT_READ_FAILED", err)
	}
	if strings.TrimSpace(input.ChapterID) == "" || strings.TrimSpace(input.PlanRevisionID) == "" || input.Plan == nil {
		return ContentRevision{}, writerPermanent("WRITER_INPUT_INVALID", ErrWriterPlanNotBound)
	}

	run, err := s.store.GetGenerationRun(ctx, job.RunID)
	if err != nil {
		if errors.Is(err, ErrGenerationRunNotFound) {
			return ContentRevision{}, writerPermanent("WRITER_RUN_MISSING", err)
		}
		return ContentRevision{}, writerTransient("WRITER_RUN_READ_FAILED", err)
	}

	// Retry/idempotency path: business output may already exist even if a prior
	// attempt failed after persistence but before output_ref/job completion.
	existing, err := store.GetWriterOutput(ctx, run.ID, input.PlanRevisionID)
	if err == nil {
		return s.recordAndVerifyWriterOutput(ctx, store, job, run, input, existing)
	}
	if !errors.Is(err, ErrContentRevisionNotFound) {
		return ContentRevision{}, writerTransient("WRITER_OUTPUT_LOOKUP_FAILED", err)
	}

	if s.textAI == nil {
		return ContentRevision{}, writerPermanent("WRITER_PROVIDER_NOT_CONFIGURED", errors.New("text AI not configured"))
	}

	prompt, err := buildWriterPrompt(run, input)
	if err != nil {
		return ContentRevision{}, writerPermanent("WRITER_INPUT_INVALID", err)
	}

	generated, err := s.textAI.GenerateText(ctx, TextAIInput{Prompt: prompt})
	if err != nil {
		return ContentRevision{}, writerTransient("WRITER_PROVIDER_FAILED", err)
	}
	contentText := strings.TrimSpace(generated.Text)
	if contentText == "" {
		return ContentRevision{}, writerTransient("WRITER_EMPTY_OUTPUT", errors.New("text provider returned empty output"))
	}

	revisionNo, err := s.store.NextContentRevision(ctx, input.ChapterID)
	if err != nil {
		return ContentRevision{}, writerTransient("WRITER_REVISION_ALLOCATE_FAILED", err)
	}

	baseCanonVersionID := strings.TrimSpace(run.BaseCanonVersionID)
	if baseCanonVersionID == "" {
		baseCanonVersionID = strings.TrimSpace(input.BaseCanonVersionID)
	}

	revision := ContentRevision{
		ID:                 uuid.NewString(),
		ChapterID:          input.ChapterID,
		RevisionNo:         revisionNo,
		ContentText:        contentText,
		SourceType:         "AI_GENERATED",
		PlanRevisionID:     input.PlanRevisionID,
		BaseCanonVersionID: baseCanonVersionID,
		GenerationRunID:    run.ID,
		Status:             "CANDIDATE",
		CreatedBy:          run.RequestedBy,
	}

	created, err := s.store.CreateContentRevision(ctx, revision)
	if err != nil {
		// A competing/retried attempt may have won the unique run/plan output
		// race. Re-read that durable output before deciding this attempt failed.
		existing, lookupErr := store.GetWriterOutput(ctx, run.ID, input.PlanRevisionID)
		if lookupErr == nil {
			return s.recordAndVerifyWriterOutput(ctx, store, job, run, input, existing)
		}
		return ContentRevision{}, writerTransient("WRITER_OUTPUT_PERSIST_FAILED", err)
	}

	return s.recordAndVerifyWriterOutput(ctx, store, job, run, input, created)
}

func buildWriterPrompt(run GenerationRun, input WriterJobInput) (string, error) {
	baseCanonVersionID := strings.TrimSpace(run.BaseCanonVersionID)
	if baseCanonVersionID == "" {
		baseCanonVersionID = strings.TrimSpace(input.BaseCanonVersionID)
	}

	payload := struct {
		StoryID            string         `json:"story_id"`
		ChapterID          string         `json:"chapter_id"`
		PlanRevisionID     string         `json:"plan_revision_id"`
		BaseCanonVersionID string         `json:"base_canon_version_id,omitempty"`
		WorkflowVersion    string         `json:"workflow_version,omitempty"`
		Plan               map[string]any `json:"plan"`
	}{
		StoryID:            run.StoryID,
		ChapterID:          input.ChapterID,
		PlanRevisionID:     input.PlanRevisionID,
		BaseCanonVersionID: baseCanonVersionID,
		WorkflowVersion:    run.WorkflowVersion,
		Plan:               input.Plan,
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode frozen writer input: %w", err)
	}
	return "Write the chapter prose from this frozen canonical input. Preserve the plan constraints and return only chapter prose.\nINPUT_JSON=" + string(encoded), nil
}

func (s *Service) recordAndVerifyWriterOutput(
	ctx context.Context,
	store WriterStore,
	job GenerationJob,
	run GenerationRun,
	input WriterJobInput,
	revision ContentRevision,
) (ContentRevision, error) {
	if err := validateWriterOutput(run, input, revision); err != nil {
		return ContentRevision{}, writerPermanent("WRITER_OUTPUT_INVALID", err)
	}

	updatedJob, err := store.UpdateWriterJobOutputRef(ctx, job.ID, map[string]any{
		"content_revision_id": revision.ID,
		"chapter_id":          revision.ChapterID,
		"plan_revision_id":    revision.PlanRevisionID,
	})
	if err != nil {
		return ContentRevision{}, writerTransient("WRITER_OUTPUT_RECORD_FAILED", err)
	}
	if outputID, _ := updatedJob.OutputRef["content_revision_id"].(string); outputID != revision.ID {
		return ContentRevision{}, writerTransient("WRITER_OUTPUT_RECORD_FAILED", errors.New("job output_ref did not persist content revision id"))
	}

	verified, err := s.store.GetContentRevision(ctx, revision.ID)
	if err != nil {
		return ContentRevision{}, writerTransient("WRITER_OUTPUT_VERIFY_FAILED", err)
	}
	if err := validateWriterOutput(run, input, verified); err != nil {
		return ContentRevision{}, writerPermanent("WRITER_OUTPUT_INVALID", err)
	}
	return verified, nil
}

func validateWriterOutput(run GenerationRun, input WriterJobInput, revision ContentRevision) error {
	if strings.TrimSpace(revision.ID) == "" {
		return errors.New("content revision id is empty")
	}
	if revision.ChapterID != input.ChapterID {
		return fmt.Errorf("content chapter %q does not match frozen chapter %q", revision.ChapterID, input.ChapterID)
	}
	if revision.PlanRevisionID != input.PlanRevisionID {
		return fmt.Errorf("content plan %q does not match frozen plan %q", revision.PlanRevisionID, input.PlanRevisionID)
	}
	if revision.GenerationRunID != run.ID {
		return fmt.Errorf("content run %q does not match generation run %q", revision.GenerationRunID, run.ID)
	}

	expectedCanon := strings.TrimSpace(run.BaseCanonVersionID)
	if expectedCanon == "" {
		expectedCanon = strings.TrimSpace(input.BaseCanonVersionID)
	}
	if expectedCanon != "" && revision.BaseCanonVersionID != expectedCanon {
		return fmt.Errorf("content base canon %q does not match frozen canon %q", revision.BaseCanonVersionID, expectedCanon)
	}
	return nil
}

func writerPermanent(code string, err error) error {
	return &ClassifiedError{Class: "PERMANENT", Code: code, Err: err}
}

func writerTransient(code string, err error) error {
	return &ClassifiedError{Class: "TRANSIENT", Code: code, Err: err}
}
