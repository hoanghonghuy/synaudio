package generation

import (
	"context"
)

type fakeStore struct {
	revisions          map[string][]ContentRevision
	approvals          map[string][]ContentApproval
	nextRev            map[string]int
	runs               map[string][]GenerationRun
	jobs               map[string][]GenerationJob
	nextAttempt        map[string]int
	attempts           map[string][]JobAttempt
	reviews            map[string][]ChapterReview
	writerInputs       map[string]WriterJobInput
	currentWriterPlans map[string]WriterJobInput
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		revisions:          map[string][]ContentRevision{},
		approvals:          map[string][]ContentApproval{},
		nextRev:            map[string]int{},
		runs:               map[string][]GenerationRun{},
		jobs:               map[string][]GenerationJob{},
		nextAttempt:        map[string]int{},
		attempts:           map[string][]JobAttempt{},
		reviews:            map[string][]ChapterReview{},
		writerInputs:       map[string]WriterJobInput{},
		currentWriterPlans: map[string]WriterJobInput{},
	}
}

func (s *fakeStore) NextContentRevision(_ context.Context, chapterID string) (int, error) {
	s.nextRev[chapterID]++
	return s.nextRev[chapterID], nil
}

func (s *fakeStore) CreateContentRevision(_ context.Context, r ContentRevision) (ContentRevision, error) {
	s.revisions[r.ChapterID] = append(s.revisions[r.ChapterID], r)
	return r, nil
}

func (s *fakeStore) GetContentRevision(_ context.Context, revisionID string) (ContentRevision, error) {
	for _, rs := range s.revisions {
		for _, r := range rs {
			if r.ID == revisionID {
				return r, nil
			}
		}
	}
	return ContentRevision{}, ErrContentRevisionNotFound
}

func (s *fakeStore) ListContentRevisions(_ context.Context, chapterID string) ([]ContentRevision, error) {
	return s.revisions[chapterID], nil
}

func (s *fakeStore) UpdateContentRevisionStatus(_ context.Context, revisionID, status string) (ContentRevision, error) {
	for chapterID, rs := range s.revisions {
		for i, r := range rs {
			if r.ID == revisionID {
				r.Status = status
				s.revisions[chapterID][i] = r
				return r, nil
			}
		}
	}
	return ContentRevision{}, ErrContentRevisionNotFound
}

func (s *fakeStore) CreateContentApproval(_ context.Context, a ContentApproval) (ContentApproval, error) {
	s.approvals[a.ChapterID] = append(s.approvals[a.ChapterID], a)
	return a, nil
}

func (s *fakeStore) CreateGenerationRun(_ context.Context, r GenerationRun) (GenerationRun, error) {
	s.runs[r.StoryID] = append(s.runs[r.StoryID], r)
	return r, nil
}

func (s *fakeStore) GetGenerationRun(_ context.Context, runID string) (GenerationRun, error) {
	for _, rs := range s.runs {
		for _, r := range rs {
			if r.ID == runID {
				return r, nil
			}
		}
	}
	return GenerationRun{}, ErrGenerationRunNotFound
}

func (s *fakeStore) CreateGenerationJob(_ context.Context, j GenerationJob) (GenerationJob, error) {
	s.jobs[j.RunID] = append(s.jobs[j.RunID], j)
	return j, nil
}

func (s *fakeStore) CreateWriterGenerationJob(_ context.Context, j GenerationJob, chapterID string) (GenerationJob, error) {
	input, ok := s.currentWriterPlans[chapterID]
	if !ok {
		// Shared queue/run tests are not plan-domain tests; give them a deterministic
		// frozen plan fixture while production pgstore requires a real current plan.
		input = WriterJobInput{
			ChapterID:          chapterID,
			PlanRevisionID:     "plan-" + chapterID,
			Plan:               map[string]any{"chapter_id": chapterID},
			BaseCanonVersionID: "canon-" + chapterID,
		}
	}
	input.JobID = j.ID
	input.ChapterID = chapterID
	planCopy := make(map[string]any, len(input.Plan))
	for key, value := range input.Plan {
		planCopy[key] = value
	}
	input.Plan = planCopy

	s.writerInputs[j.ID] = input
	s.jobs[j.RunID] = append(s.jobs[j.RunID], j)
	return j, nil
}

func (s *fakeStore) GetWriterJobInput(_ context.Context, jobID string) (WriterJobInput, error) {
	input, ok := s.writerInputs[jobID]
	if !ok {
		return WriterJobInput{}, ErrWriterJobInputNotFound
	}
	return input, nil
}

func (s *fakeStore) GetWriterOutput(_ context.Context, runID, planRevisionID string) (ContentRevision, error) {
	for _, revisions := range s.revisions {
		for _, revision := range revisions {
			if revision.GenerationRunID == runID && revision.PlanRevisionID == planRevisionID && revision.SourceType == "AI_GENERATED" {
				return revision, nil
			}
		}
	}
	return ContentRevision{}, ErrContentRevisionNotFound
}

func (s *fakeStore) UpdateWriterJobOutputRef(_ context.Context, jobID string, outputRef map[string]any) (GenerationJob, error) {
	for runID, jobs := range s.jobs {
		for i, job := range jobs {
			if job.ID == jobID {
				job.OutputRef = outputRef
				s.jobs[runID][i] = job
				return job, nil
			}
		}
	}
	return GenerationJob{}, ErrGenerationJobNotFound
}

func (s *fakeStore) NextAttemptNo(_ context.Context, jobID string) (int, error) {
	s.nextAttempt[jobID]++
	return s.nextAttempt[jobID], nil
}

func (s *fakeStore) CreateJobAttempt(_ context.Context, a JobAttempt) (JobAttempt, error) {
	s.attempts[a.JobID] = append(s.attempts[a.JobID], a)
	return a, nil
}

func (s *fakeStore) CreateChapterReview(_ context.Context, r ChapterReview) (ChapterReview, error) {
	s.reviews[r.ChapterID] = append(s.reviews[r.ChapterID], r)
	return r, nil
}

func (s *fakeStore) ListChapterReviews(_ context.Context, chapterID string) ([]ChapterReview, error) {
	return s.reviews[chapterID], nil
}

func (s *fakeStore) ClaimNextJob(_ context.Context, workerID string) (GenerationJob, error) {
	for runID, jobs := range s.jobs {
		for i, j := range jobs {
			if j.Status == "PENDING" {
				j.Status = "RUNNING"
				j.LockedBy = workerID
				s.jobs[runID][i] = j
				return j, nil
			}
		}
	}
	return GenerationJob{}, ErrNoRunnableJob
}

func (s *fakeStore) UpdateJobStatus(_ context.Context, jobID, status, errorClass, errorCode string) (GenerationJob, error) {
	for runID, jobs := range s.jobs {
		for i, j := range jobs {
			if j.ID == jobID {
				j.Status = status
				j.LastErrorClass = errorClass
				j.LastErrorCode = errorCode
				s.jobs[runID][i] = j
				return j, nil
			}
		}
	}
	return GenerationJob{}, ErrGenerationJobNotFound
}

func (s *fakeStore) UpdateJobAttemptStatus(_ context.Context, attemptID, status, errorClass, errorCode string) (JobAttempt, error) {
	return JobAttempt{ID: attemptID, Status: status, ErrorClass: errorClass, ErrorCode: errorCode}, nil
}

func (s *fakeStore) ReclaimStaleJobs(_ context.Context, olderThan string) ([]GenerationJob, error) {
	out := []GenerationJob{}
	for runID, jobs := range s.jobs {
		for i, j := range jobs {
			if j.Status == "RUNNING" {
				j.Status = "PENDING"
				j.LockedBy = ""
				s.jobs[runID][i] = j
				out = append(out, j)
			}
		}
	}
	return out, nil
}

func (s *fakeStore) CancelJob(_ context.Context, jobID string) (GenerationJob, error) {
	for runID, jobs := range s.jobs {
		for i, j := range jobs {
			if j.ID == jobID {
				j.Status = "CANCELLED"
				s.jobs[runID][i] = j
				return j, nil
			}
		}
	}
	return GenerationJob{}, ErrGenerationJobNotFound
}

func (s *fakeStore) ListJobsByRun(_ context.Context, runID string) ([]GenerationJob, error) {
	return s.jobs[runID], nil
}

func (s *fakeStore) UpdateJobAttemptUsage(_ context.Context, attemptID string, usage map[string]any) (JobAttempt, error) {
	for jobID, as := range s.attempts {
		for i, a := range as {
			if a.ID == attemptID {
				a.Usage = usage
				s.attempts[jobID][i] = a
				return a, nil
			}
		}
	}
	return JobAttempt{}, ErrJobAttemptNotFound
}

func (s *fakeStore) ListUsageByStory(_ context.Context, storyID string) ([]JobAttempt, error) {
	out := []JobAttempt{}
	for _, as := range s.attempts {
		for _, a := range as {
			if a.Usage != nil {
				out = append(out, a)
			}
		}
	}
	return out, nil
}
