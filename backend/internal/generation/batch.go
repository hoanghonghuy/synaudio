package generation

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// WriterBatchJob is one WRITER job plus the chapter whose current plan must be
// frozen during batch establishment.
type WriterBatchJob struct {
	Job       GenerationJob
	ChapterID string
}

// WriterBatchStore establishes a CHAPTER_GENERATION run and every frozen
// WRITER input atomically. Implementations must not expose any job to workers
// unless the complete batch commits successfully.
type WriterBatchStore interface {
	EstablishWriterBatch(ctx context.Context, run GenerationRun, jobs []WriterBatchJob) (GenerationRun, error)
}

// StartBatchGeneration creates a GenerationRun of type CHAPTER_GENERATION and
// enqueues sequential WRITER jobs, one per chapter, in order. Each job freezes
// its own exact Chapter Plan revision because one batch run can span many chapters.
func (s *Service) StartBatchGeneration(ctx context.Context, storyID string, chapterIDs []string, requestedBy string) (GenerationRun, error) {
	if len(chapterIDs) == 0 {
		return GenerationRun{}, errors.New("chapter ids must not be empty")
	}

	batchStore, ok := s.store.(WriterBatchStore)
	if !ok {
		return GenerationRun{}, errors.New("atomic writer batch persistence not configured")
	}

	run := GenerationRun{
		ID:          uuid.NewString(),
		RunType:     "CHAPTER_GENERATION",
		StoryID:     storyID,
		Status:      "PENDING",
		RequestedBy: requestedBy,
	}

	jobs := make([]WriterBatchJob, 0, len(chapterIDs))
	for _, rawChapterID := range chapterIDs {
		chapterID := strings.TrimSpace(rawChapterID)
		if chapterID == "" {
			return GenerationRun{}, ErrWriterPlanNotBound
		}
		jobs = append(jobs, WriterBatchJob{
			Job: GenerationJob{
				ID:          uuid.NewString(),
				RunID:       run.ID,
				JobType:     "WRITER",
				Status:      "PENDING",
				MaxAttempts: 3,
			},
			ChapterID: chapterID,
		})
	}

	return batchStore.EstablishWriterBatch(ctx, run, jobs)
}

// MarkDownstreamStale marks all jobs after the given chapter as STALE.
// This is used when an earlier chapter changes, invalidating downstream candidates.
func (s *Service) MarkDownstreamStale(ctx context.Context, runID, chapterID string) ([]GenerationJob, error) {
	jobs, err := s.store.ListJobsByRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	writerStore, err := s.writerStore()
	if err != nil {
		return nil, err
	}

	var stale []GenerationJob
	found := false
	for _, j := range jobs {
		if j.JobType != "WRITER" {
			continue
		}
		input, err := writerStore.GetWriterJobInput(ctx, j.ID)
		if err != nil {
			return nil, err
		}
		if input.ChapterID == chapterID {
			found = true
			continue
		}
		if found && j.Status != "STALE" && j.Status != "SUCCEEDED" {
			updated, err := s.store.UpdateJobStatus(ctx, j.ID, "STALE", "", "")
			if err != nil {
				return nil, err
			}
			stale = append(stale, updated)
		}
	}

	return stale, nil
}
