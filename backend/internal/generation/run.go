package generation

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrGenerationRunNotFound = errors.New("generation run not found")
)

// GenerationRun is a high-level workflow intent.
type GenerationRun struct {
	ID                string
	RunType           string
	StoryID           string
	ChapterID         string
	Status            string
	WaitingReason     string
	WorkflowVersion   string
	Priority          int
	BaseCanonVersionID string
	ContextSnapshotID string
	RequestedBy       string
	IdempotencyKey    string
}

// GenerationJob is a smaller execution unit inside a GenerationRun.
type GenerationJob struct {
	ID               string
	RunID            string
	JobType          string
	Status           string
	Priority         int
	AttemptCount     int
	MaxAttempts      int
	InputFingerprint string
	LastErrorClass   string
	LastErrorCode    string
	OutputRef        map[string]any
}

// JobAttempt is a single execution attempt of a GenerationJob.
type JobAttempt struct {
	ID         string
	JobID      string
	AttemptNo  int
	Provider   string
	Model      string
	Status     string
	ErrorClass string
	ErrorCode  string
	Usage      map[string]any
	LatencyMs  int
}

// CreateGenerationRun creates a new PENDING generation run.
func (s *Service) CreateGenerationRun(ctx context.Context, runType, storyID, chapterID, requestedBy string) (GenerationRun, error) {
	runType = strings.TrimSpace(runType)
	if runType == "" {
		return GenerationRun{}, errors.New("run type must not be empty")
	}

	r := GenerationRun{
		ID:          uuid.NewString(),
		RunType:     runType,
		StoryID:     storyID,
		ChapterID:   chapterID,
		Status:      "PENDING",
		RequestedBy: requestedBy,
	}

	return s.store.CreateGenerationRun(ctx, r)
}

// GetGenerationRun returns a single run by ID.
func (s *Service) GetGenerationRun(ctx context.Context, runID string) (GenerationRun, error) {
	return s.store.GetGenerationRun(ctx, runID)
}

// CreateGenerationJob creates a new PENDING job within a run.
func (s *Service) CreateGenerationJob(ctx context.Context, runID, jobType string, maxAttempts int) (GenerationJob, error) {
	jobType = strings.TrimSpace(jobType)
	if jobType == "" {
		return GenerationJob{}, errors.New("job type must not be empty")
	}
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	j := GenerationJob{
		ID:          uuid.NewString(),
		RunID:       runID,
		JobType:     jobType,
		Status:      "PENDING",
		MaxAttempts: maxAttempts,
	}

	return s.store.CreateGenerationJob(ctx, j)
}

// CreateJobAttempt records a new attempt for a job.
func (s *Service) CreateJobAttempt(ctx context.Context, jobID, provider, model string) (JobAttempt, error) {
	attemptNo, err := s.store.NextAttemptNo(ctx, jobID)
	if err != nil {
		return JobAttempt{}, err
	}

	a := JobAttempt{
		ID:        uuid.NewString(),
		JobID:     jobID,
		AttemptNo: attemptNo,
		Provider:  provider,
		Model:     model,
		Status:    "RUNNING",
	}

	return s.store.CreateJobAttempt(ctx, a)
}
