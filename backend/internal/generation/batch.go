package generation

import (
	"context"
	"errors"
)

// StartBatchGeneration creates a GenerationRun of type CHAPTER_GENERATION and
// enqueues sequential WRITER jobs, one per chapter, in order. Each job freezes
// its own exact Chapter Plan revision because one batch run can span many chapters.
func (s *Service) StartBatchGeneration(ctx context.Context, storyID string, chapterIDs []string, requestedBy string) (GenerationRun, error) {
	if len(chapterIDs) == 0 {
		return GenerationRun{}, errors.New("chapter ids must not be empty")
	}

	run, err := s.CreateGenerationRun(ctx, "CHAPTER_GENERATION", storyID, "", requestedBy)
	if err != nil {
		return GenerationRun{}, err
	}

	for _, chapterID := range chapterIDs {
		if _, err := s.createWriterGenerationJob(ctx, run.ID, chapterID, 3); err != nil {
			return GenerationRun{}, err
		}
	}

	return run, nil
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
