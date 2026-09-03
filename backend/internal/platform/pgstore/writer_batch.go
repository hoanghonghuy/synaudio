package pgstore

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

type transactionBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

// EstablishWriterBatch persists the run and every WRITER job/input binding in a
// single database transaction. A missing plan or any later persistence failure
// rolls the whole request back, so workers can never observe a partial batch.
func (s *GenerationStore) EstablishWriterBatch(ctx context.Context, run generation.GenerationRun, jobs []generation.WriterBatchJob) (generation.GenerationRun, error) {
	beginner, ok := s.q.DBTX().(transactionBeginner)
	if !ok {
		return generation.GenerationRun{}, errors.New("generation store transaction support unavailable")
	}

	tx, err := beginner.Begin(ctx)
	if err != nil {
		return generation.GenerationRun{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.q.WithTx(tx)
	row, err := qtx.CreateGenerationRun(ctx, db.CreateGenerationRunParams{
		ID:                 toUUID(run.ID),
		RunType:            run.RunType,
		StoryID:            toUUID(run.StoryID),
		ChapterID:          toUUID(run.ChapterID),
		Status:             run.Status,
		WaitingReason:      toText(run.WaitingReason),
		WorkflowVersion:    toText(run.WorkflowVersion),
		Priority:           int32(run.Priority),
		BaseCanonVersionID: toUUID(run.BaseCanonVersionID),
		ContextSnapshotID:  toUUID(run.ContextSnapshotID),
		RequestedBy:        toUUID(run.RequestedBy),
		IdempotencyKey:     toText(run.IdempotencyKey),
	})
	if err != nil {
		return generation.GenerationRun{}, err
	}
	createdRun := toGenerationRun(row)

	for _, item := range jobs {
		job := item.Job
		_, err := qtx.CreateWriterGenerationJob(ctx, db.CreateWriterGenerationJobParams{
			ID:               toUUID(job.ID),
			RunID:            toUUID(job.RunID),
			JobType:          job.JobType,
			Status:           job.Status,
			Priority:         int32(job.Priority),
			InputFingerprint: toText(job.InputFingerprint),
			AttemptCount:     int32(job.AttemptCount),
			MaxAttempts:      int32(job.MaxAttempts),
			ChapterID:        toUUID(item.ChapterID),
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return generation.GenerationRun{}, generation.ErrWriterPlanNotBound
			}
			return generation.GenerationRun{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return generation.GenerationRun{}, err
	}
	return createdRun, nil
}
