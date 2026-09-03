package pgstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

func (s *GenerationStore) CreateWriterGenerationJob(ctx context.Context, j generation.GenerationJob, chapterID string) (generation.GenerationJob, error) {
	row, err := s.q.CreateWriterGenerationJob(ctx, db.CreateWriterGenerationJobParams{
		ID:               toUUID(j.ID),
		RunID:            toUUID(j.RunID),
		JobType:          j.JobType,
		Status:           j.Status,
		Priority:         int32(j.Priority),
		InputFingerprint: toText(j.InputFingerprint),
		AttemptCount:     int32(j.AttemptCount),
		MaxAttempts:      int32(j.MaxAttempts),
		ChapterID:        toUUID(chapterID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generation.GenerationJob{}, generation.ErrWriterPlanNotBound
		}
		return generation.GenerationJob{}, err
	}
	return toGenerationJob(row), nil
}

func (s *GenerationStore) GetWriterJobInput(ctx context.Context, jobID string) (generation.WriterJobInput, error) {
	row, err := s.q.GetWriterJobInput(ctx, toUUID(jobID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generation.WriterJobInput{}, generation.ErrWriterJobInputNotFound
		}
		return generation.WriterJobInput{}, err
	}

	var plan map[string]any
	if err := json.Unmarshal(row.Plan, &plan); err != nil {
		return generation.WriterJobInput{}, fmt.Errorf("decode frozen writer plan: %w", err)
	}

	return generation.WriterJobInput{
		JobID:              fromUUID(row.JobID),
		ChapterID:          fromUUID(row.ChapterID),
		PlanRevisionID:     fromUUID(row.PlanRevisionID),
		Plan:               plan,
		BaseCanonVersionID: fromUUID(row.BaseCanonVersionID),
	}, nil
}

func (s *GenerationStore) GetWriterOutput(ctx context.Context, runID, planRevisionID string) (generation.ContentRevision, error) {
	row, err := s.q.GetWriterOutput(ctx, db.GetWriterOutputParams{
		GenerationRunID: toUUID(runID),
		PlanRevisionID:  toUUID(planRevisionID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generation.ContentRevision{}, generation.ErrContentRevisionNotFound
		}
		return generation.ContentRevision{}, err
	}
	return toContentRevision(row), nil
}

func (s *GenerationStore) UpdateWriterJobOutputRef(ctx context.Context, jobID string, outputRef map[string]any) (generation.GenerationJob, error) {
	encoded, err := json.Marshal(outputRef)
	if err != nil {
		return generation.GenerationJob{}, fmt.Errorf("encode writer output ref: %w", err)
	}

	row, err := s.q.UpdateWriterJobOutputRef(ctx, db.UpdateWriterJobOutputRefParams{
		ID:        toUUID(jobID),
		OutputRef: encoded,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generation.GenerationJob{}, generation.ErrGenerationJobNotFound
		}
		return generation.GenerationJob{}, err
	}
	return toGenerationJob(row), nil
}
