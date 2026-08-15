package pgstore

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

// GenerationStore implements generation.Store backed by PostgreSQL via sqlc.
type GenerationStore struct {
	q *db.Queries
}

func NewGenerationStore(q *db.Queries) *GenerationStore {
	return &GenerationStore{q: q}
}

// ============================================================
// Content Revisions
// ============================================================

func (s *GenerationStore) NextContentRevision(ctx context.Context, chapterID string) (int, error) {
	n, err := s.q.NextContentRevision(ctx, toUUID(chapterID))
	return int(n), err
}

func (s *GenerationStore) CreateContentRevision(ctx context.Context, r generation.ContentRevision) (generation.ContentRevision, error) {
	row, err := s.q.CreateContentRevision(ctx, db.CreateContentRevisionParams{
		ID:                 toUUID(r.ID),
		ChapterID:          toUUID(r.ChapterID),
		RevisionNo:         int32(r.RevisionNo),
		ContentText:        r.ContentText,
		SourceType:         r.SourceType,
		BasedOnRevisionID:  toUUID(r.BasedOnRevisionID),
		PlanRevisionID:     toUUID(r.PlanRevisionID),
		BaseCanonVersionID: toUUID(r.BaseCanonVersionID),
		GenerationRunID:    toUUID(r.GenerationRunID),
		Status:             r.Status,
		CreatedBy:          toUUID(r.CreatedBy),
	})
	if err != nil {
		return generation.ContentRevision{}, err
	}
	return toContentRevision(row), nil
}

func (s *GenerationStore) GetContentRevision(ctx context.Context, revisionID string) (generation.ContentRevision, error) {
	row, err := s.q.GetContentRevision(ctx, toUUID(revisionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generation.ContentRevision{}, generation.ErrContentRevisionNotFound
		}
		return generation.ContentRevision{}, err
	}
	return toContentRevision(row), nil
}

func (s *GenerationStore) ListContentRevisions(ctx context.Context, chapterID string) ([]generation.ContentRevision, error) {
	rows, err := s.q.ListContentRevisions(ctx, toUUID(chapterID))
	if err != nil {
		return nil, err
	}
	out := make([]generation.ContentRevision, 0, len(rows))
	for _, r := range rows {
		out = append(out, toContentRevision(r))
	}
	return out, nil
}

func (s *GenerationStore) CreateContentApproval(ctx context.Context, a generation.ContentApproval) (generation.ContentApproval, error) {
	warnings, _ := json.Marshal(a.WarningsSnapshot)
	override, _ := json.Marshal(a.OverrideSnapshot)
	row, err := s.q.CreateContentApproval(ctx, db.CreateContentApprovalParams{
		ID:                toUUID(a.ID),
		ChapterID:         toUUID(a.ChapterID),
		ContentRevisionID: toUUID(a.ContentRevisionID),
		ApprovedBy:        toUUID(a.ApprovedBy),
		WarningsSnapshot:  warnings,
		OverrideSnapshot:  override,
	})
	if err != nil {
		return generation.ContentApproval{}, err
	}
	return toContentApproval(row), nil
}

// ============================================================
// Generation Runs / Jobs / Attempts
// ============================================================

func (s *GenerationStore) CreateGenerationRun(ctx context.Context, r generation.GenerationRun) (generation.GenerationRun, error) {
	row, err := s.q.CreateGenerationRun(ctx, db.CreateGenerationRunParams{
		ID:                 toUUID(r.ID),
		RunType:            r.RunType,
		StoryID:            toUUID(r.StoryID),
		ChapterID:          toUUID(r.ChapterID),
		Status:             r.Status,
		WaitingReason:      toText(r.WaitingReason),
		WorkflowVersion:    toText(r.WorkflowVersion),
		Priority:           int32(r.Priority),
		BaseCanonVersionID: toUUID(r.BaseCanonVersionID),
		ContextSnapshotID:  toUUID(r.ContextSnapshotID),
		RequestedBy:        toUUID(r.RequestedBy),
		IdempotencyKey:     toText(r.IdempotencyKey),
	})
	if err != nil {
		return generation.GenerationRun{}, err
	}
	return toGenerationRun(row), nil
}

func (s *GenerationStore) GetGenerationRun(ctx context.Context, runID string) (generation.GenerationRun, error) {
	row, err := s.q.GetGenerationRun(ctx, toUUID(runID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generation.GenerationRun{}, generation.ErrGenerationRunNotFound
		}
		return generation.GenerationRun{}, err
	}
	return toGenerationRun(row), nil
}

func (s *GenerationStore) CreateGenerationJob(ctx context.Context, j generation.GenerationJob) (generation.GenerationJob, error) {
	row, err := s.q.CreateGenerationJob(ctx, db.CreateGenerationJobParams{
		ID:               toUUID(j.ID),
		RunID:            toUUID(j.RunID),
		JobType:          j.JobType,
		Status:           j.Status,
		Priority:         int32(j.Priority),
		InputFingerprint: toText(j.InputFingerprint),
		AttemptCount:     int32(j.AttemptCount),
		MaxAttempts:      int32(j.MaxAttempts),
	})
	if err != nil {
		return generation.GenerationJob{}, err
	}
	return toGenerationJob(row), nil
}

func (s *GenerationStore) NextAttemptNo(ctx context.Context, jobID string) (int, error) {
	n, err := s.q.NextAttemptNo(ctx, toUUID(jobID))
	return int(n), err
}

func (s *GenerationStore) CreateJobAttempt(ctx context.Context, a generation.JobAttempt) (generation.JobAttempt, error) {
	row, err := s.q.CreateJobAttempt(ctx, db.CreateJobAttemptParams{
		ID:        toUUID(a.ID),
		JobID:     toUUID(a.JobID),
		AttemptNo: int32(a.AttemptNo),
		Provider:  toText(a.Provider),
		Model:     toText(a.Model),
		Status:    a.Status,
	})
	if err != nil {
		return generation.JobAttempt{}, err
	}
	return toJobAttempt(row), nil
}

func (s *GenerationStore) ClaimNextJob(ctx context.Context, workerID string) (generation.GenerationJob, error) {
	row, err := s.q.ClaimNextJob(ctx, toText(workerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generation.GenerationJob{}, generation.ErrNoRunnableJob
		}
		return generation.GenerationJob{}, err
	}
	return toGenerationJob(row), nil
}

func (s *GenerationStore) UpdateJobStatus(ctx context.Context, jobID, status, errorClass, errorCode string) (generation.GenerationJob, error) {
	row, err := s.q.UpdateJobStatus(ctx, db.UpdateJobStatusParams{
		ID:             toUUID(jobID),
		Status:         status,
		LastErrorClass: toText(errorClass),
		LastErrorCode:  toText(errorCode),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generation.GenerationJob{}, generation.ErrGenerationJobNotFound
		}
		return generation.GenerationJob{}, err
	}
	return toGenerationJob(row), nil
}

func (s *GenerationStore) UpdateJobAttemptStatus(ctx context.Context, attemptID, status, errorClass, errorCode string) (generation.JobAttempt, error) {
	row, err := s.q.UpdateJobAttemptStatus(ctx, db.UpdateJobAttemptStatusParams{
		ID:         toUUID(attemptID),
		Status:     status,
		ErrorClass: toText(errorClass),
		ErrorCode:  toText(errorCode),
	})
	if err != nil {
		return generation.JobAttempt{}, err
	}
	return toJobAttempt(row), nil
}

func (s *GenerationStore) ReclaimStaleJobs(ctx context.Context, olderThan string) ([]generation.GenerationJob, error) {
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

func (s *GenerationStore) CancelJob(ctx context.Context, jobID string) (generation.GenerationJob, error) {
	row, err := s.q.CancelJob(ctx, toUUID(jobID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generation.GenerationJob{}, generation.ErrGenerationJobNotFound
		}
		return generation.GenerationJob{}, err
	}
	return toGenerationJob(row), nil
}

// ============================================================
// Chapter Reviews
// ============================================================

func (s *GenerationStore) CreateChapterReview(ctx context.Context, r generation.ChapterReview) (generation.ChapterReview, error) {
	report, _ := json.Marshal(r.Report)
	row, err := s.q.CreateChapterReview(ctx, db.CreateChapterReviewParams{
		ID:                toUUID(r.ID),
		ChapterID:         toUUID(r.ChapterID),
		ContentRevisionID: toUUID(r.ContentRevisionID),
		ReviewType:        r.ReviewType,
		Outcome:           r.Outcome,
		Report:            report,
	})
	if err != nil {
		return generation.ChapterReview{}, err
	}
	return toChapterReview(row), nil
}

func (s *GenerationStore) ListChapterReviews(ctx context.Context, chapterID string) ([]generation.ChapterReview, error) {
	rows, err := s.q.ListChapterReviews(ctx, toUUID(chapterID))
	if err != nil {
		return nil, err
	}
	out := make([]generation.ChapterReview, 0, len(rows))
	for _, r := range rows {
		out = append(out, toChapterReview(r))
	}
	return out, nil
}

// ============================================================
// Converters
// ============================================================

func toContentRevision(row db.ChapterContentRevision) generation.ContentRevision {
	return generation.ContentRevision{
		ID:                fromUUID(row.ID),
		ChapterID:         fromUUID(row.ChapterID),
		RevisionNo:        int(row.RevisionNo),
		ContentText:       row.ContentText,
		SourceType:        row.SourceType,
		BasedOnRevisionID: fromUUID(row.BasedOnRevisionID),
		PlanRevisionID:    fromUUID(row.PlanRevisionID),
		BaseCanonVersionID: fromUUID(row.BaseCanonVersionID),
		GenerationRunID:   fromUUID(row.GenerationRunID),
		Status:            row.Status,
		CreatedBy:         fromUUID(row.CreatedBy),
	}
}

func toContentApproval(row db.ContentApproval) generation.ContentApproval {
	var warnings map[string]any
	var override map[string]any
	_ = json.Unmarshal(row.WarningsSnapshot, &warnings)
	_ = json.Unmarshal(row.OverrideSnapshot, &override)
	return generation.ContentApproval{
		ID:                fromUUID(row.ID),
		ChapterID:         fromUUID(row.ChapterID),
		ContentRevisionID: fromUUID(row.ContentRevisionID),
		ApprovedBy:        fromUUID(row.ApprovedBy),
		WarningsSnapshot:  warnings,
		OverrideSnapshot:  override,
	}
}

func toGenerationRun(row db.GenerationRun) generation.GenerationRun {
	return generation.GenerationRun{
		ID:                fromUUID(row.ID),
		RunType:           row.RunType,
		StoryID:           fromUUID(row.StoryID),
		ChapterID:         fromUUID(row.ChapterID),
		Status:            row.Status,
		WaitingReason:     fromText(row.WaitingReason),
		WorkflowVersion:   fromText(row.WorkflowVersion),
		Priority:          int(row.Priority),
		BaseCanonVersionID: fromUUID(row.BaseCanonVersionID),
		ContextSnapshotID: fromUUID(row.ContextSnapshotID),
		RequestedBy:       fromUUID(row.RequestedBy),
		IdempotencyKey:    fromText(row.IdempotencyKey),
	}
}

func toGenerationJob(row db.GenerationJob) generation.GenerationJob {
	var outputRef map[string]any
	_ = json.Unmarshal(row.OutputRef, &outputRef)
	return generation.GenerationJob{
		ID:               fromUUID(row.ID),
		RunID:            fromUUID(row.RunID),
		JobType:          row.JobType,
		Status:           row.Status,
		Priority:         int(row.Priority),
		AttemptCount:     int(row.AttemptCount),
		MaxAttempts:      int(row.MaxAttempts),
		InputFingerprint: fromText(row.InputFingerprint),
		LastErrorClass:   fromText(row.LastErrorClass),
		LastErrorCode:    fromText(row.LastErrorCode),
		LockedBy:         fromText(row.LockedBy),
		OutputRef:        outputRef,
	}
}

func toJobAttempt(row db.GenerationJobAttempt) generation.JobAttempt {
	var usage map[string]any
	_ = json.Unmarshal(row.Usage, &usage)
	return generation.JobAttempt{
		ID:         fromUUID(row.ID),
		JobID:      fromUUID(row.JobID),
		AttemptNo:  int(row.AttemptNo),
		Provider:   fromText(row.Provider),
		Model:      fromText(row.Model),
		Status:     row.Status,
		ErrorClass: fromText(row.ErrorClass),
		ErrorCode:  fromText(row.ErrorCode),
		Usage:      usage,
		LatencyMs:  int(row.LatencyMs.Int32),
	}
}

func toChapterReview(row db.ChapterReview) generation.ChapterReview {
	var report map[string]any
	_ = json.Unmarshal(row.Report, &report)
	return generation.ChapterReview{
		ID:                fromUUID(row.ID),
		ChapterID:         fromUUID(row.ChapterID),
		ContentRevisionID: fromUUID(row.ContentRevisionID),
		ReviewType:        row.ReviewType,
		Outcome:           row.Outcome,
		Report:            report,
	}
}
