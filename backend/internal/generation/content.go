package generation

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrContentRevisionNotFound = errors.New("content revision not found")
)

// ContentRevision is a versioned Chapter prose revision.
type ContentRevision struct {
	ID                string
	ChapterID         string
	RevisionNo        int
	ContentText       string
	SourceType        string
	BasedOnRevisionID string
	PlanRevisionID    string
	BaseCanonVersionID string
	GenerationRunID   string
	Status            string
	CreatedBy         string
}

// ContentApproval is an append-only record of Admin content approval.
type ContentApproval struct {
	ID                string
	ChapterID         string
	ContentRevisionID string
	ApprovedBy        string
	WarningsSnapshot  map[string]any
	OverrideSnapshot  map[string]any
}

// Store is the persistence boundary for the generation service.
type Store interface {
	NextContentRevision(ctx context.Context, chapterID string) (int, error)
	CreateContentRevision(ctx context.Context, r ContentRevision) (ContentRevision, error)
	GetContentRevision(ctx context.Context, revisionID string) (ContentRevision, error)
	ListContentRevisions(ctx context.Context, chapterID string) ([]ContentRevision, error)
	UpdateContentRevisionStatus(ctx context.Context, revisionID, status string) (ContentRevision, error)
	CreateContentApproval(ctx context.Context, a ContentApproval) (ContentApproval, error)

	CreateGenerationRun(ctx context.Context, r GenerationRun) (GenerationRun, error)
	GetGenerationRun(ctx context.Context, runID string) (GenerationRun, error)
	CreateGenerationJob(ctx context.Context, j GenerationJob) (GenerationJob, error)
	NextAttemptNo(ctx context.Context, jobID string) (int, error)
	CreateJobAttempt(ctx context.Context, a JobAttempt) (JobAttempt, error)

	CreateChapterReview(ctx context.Context, r ChapterReview) (ChapterReview, error)
	ListChapterReviews(ctx context.Context, chapterID string) ([]ChapterReview, error)

	ClaimNextJob(ctx context.Context, workerID string) (GenerationJob, error)
	UpdateJobStatus(ctx context.Context, jobID, status, errorClass, errorCode string) (GenerationJob, error)
	UpdateJobAttemptStatus(ctx context.Context, attemptID, status, errorClass, errorCode string) (JobAttempt, error)
	ReclaimStaleJobs(ctx context.Context, olderThan string) ([]GenerationJob, error)
	CancelJob(ctx context.Context, jobID string) (GenerationJob, error)
	ListJobsByRun(ctx context.Context, runID string) ([]GenerationJob, error)

	UpdateJobAttemptUsage(ctx context.Context, attemptID string, usage map[string]any) (JobAttempt, error)
	ListUsageByStory(ctx context.Context, storyID string) ([]JobAttempt, error)
}

// Service orchestrates Chapter content generation and review.
type Service struct {
	store            Store
	textAI           TextAIProvider
	durationAnalyzer DurationAnalyzer
	reviewer         Reviewer
}

type Option func(*Service)

func WithTextAI(p TextAIProvider) Option {
	return func(svc *Service) {
		svc.textAI = p
	}
}

func WithDurationAnalyzer(a DurationAnalyzer) Option {
	return func(svc *Service) {
		svc.durationAnalyzer = a
	}
}

func WithReviewer(r Reviewer) Option {
	return func(svc *Service) {
		svc.reviewer = r
	}
}

func NewService(store Store, opts ...Option) *Service {
	svc := &Service{store: store}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// CreateContentRevision creates a new CANDIDATE content revision.
func (s *Service) CreateContentRevision(ctx context.Context, chapterID, contentText, sourceType, createdBy string) (ContentRevision, error) {
	contentText = strings.TrimSpace(contentText)
	if contentText == "" {
		return ContentRevision{}, errors.New("content text must not be empty")
	}
	if sourceType == "" {
		sourceType = "AI_GENERATED"
	}

	revisionNo, err := s.store.NextContentRevision(ctx, chapterID)
	if err != nil {
		return ContentRevision{}, err
	}

	r := ContentRevision{
		ID:          uuid.NewString(),
		ChapterID:   chapterID,
		RevisionNo:  revisionNo,
		ContentText: contentText,
		SourceType:  sourceType,
		Status:      "CANDIDATE",
		CreatedBy:   createdBy,
	}

	return s.store.CreateContentRevision(ctx, r)
}

// ApproveContent records an Admin approval for an exact content revision.
func (s *Service) ApproveContent(ctx context.Context, chapterID, revisionID, approvedBy string) (ContentApproval, error) {
	if _, err := s.store.GetContentRevision(ctx, revisionID); err != nil {
		return ContentApproval{}, err
	}

	a := ContentApproval{
		ID:                uuid.NewString(),
		ChapterID:         chapterID,
		ContentRevisionID: revisionID,
		ApprovedBy:        approvedBy,
	}

	return s.store.CreateContentApproval(ctx, a)
}

// ListContentRevisions returns all revisions for a chapter, ordered by number.
func (s *Service) ListContentRevisions(ctx context.Context, chapterID string) ([]ContentRevision, error) {
	return s.store.ListContentRevisions(ctx, chapterID)
}
