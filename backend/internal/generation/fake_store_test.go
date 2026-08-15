package generation

import (
	"context"
)

type fakeStore struct {
	revisions map[string][]ContentRevision
	approvals map[string][]ContentApproval
	nextRev   map[string]int
	runs      map[string][]GenerationRun
	jobs      map[string][]GenerationJob
	nextAttempt map[string]int
	reviews   map[string][]ChapterReview
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		revisions: map[string][]ContentRevision{},
		approvals: map[string][]ContentApproval{},
		nextRev:   map[string]int{},
		runs:      map[string][]GenerationRun{},
		jobs:      map[string][]GenerationJob{},
		nextAttempt: map[string]int{},
		reviews:   map[string][]ChapterReview{},
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

func (s *fakeStore) NextAttemptNo(_ context.Context, jobID string) (int, error) {
	s.nextAttempt[jobID]++
	return s.nextAttempt[jobID], nil
}

func (s *fakeStore) CreateJobAttempt(_ context.Context, a JobAttempt) (JobAttempt, error) {
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
