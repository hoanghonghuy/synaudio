package generation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRetryJobEndpointRejectsNonRetryableJob(t *testing.T) {
	store := newFakeStore()
	run, _ := store.CreateGenerationRun(context.Background(), GenerationRun{ID: "run-1", RunType: "CHAPTER_GENERATION", StoryID: "s1"})
	store.jobs[run.ID] = []GenerationJob{{
		ID:             "job-1",
		RunID:          run.ID,
		JobType:        "WRITER",
		Status:         "FAILED",
		AttemptCount:   3,
		MaxAttempts:    3,
		LastErrorClass: "RETRY_EXHAUSTED",
		LastErrorCode:  "MAX_ATTEMPTS_EXHAUSTED",
	}}

	handler := NewHandler(NewService(store), func(context.Context, *http.Request) (string, error) {
		return "admin-1", nil
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/generation-jobs/job-1/retry", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRetryJobEndpointRequeuesRetryableJob(t *testing.T) {
	store := newFakeStore()
	store.currentWriterPlans["c1"] = WriterJobInput{ChapterID: "c1", PlanRevisionID: "plan-1", Plan: map[string]any{}}
	run, _ := store.CreateGenerationRun(context.Background(), GenerationRun{ID: "run-1", RunType: "CHAPTER_GENERATION", StoryID: "s1"})
	job := GenerationJob{
		ID:             "job-1",
		RunID:          run.ID,
		JobType:        "WRITER",
		Status:         "FAILED",
		AttemptCount:   2,
		MaxAttempts:    3,
		LastErrorClass: "TRANSIENT",
		LastErrorCode:  "PROVIDER_TIMEOUT",
	}
	store.jobs[run.ID] = []GenerationJob{job}
	store.writerInputs[job.ID] = store.currentWriterPlans["c1"]

	handler := NewHandler(NewService(store), func(context.Context, *http.Request) (string, error) {
		return "admin-1", nil
	})
	req := httptest.NewRequest(http.MethodPost, "/admin/generation-jobs/job-1/retry", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var view JobView
	if err := json.NewDecoder(rec.Body).Decode(&view); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if view.Observation != "queued" || view.Retryable {
		t.Fatalf("unexpected view: %#v", view)
	}
}

func TestChapterContentProjectionIncludesGenerationJob(t *testing.T) {
	store := newFakeStore()
	store.currentWriterPlans["chapter-1"] = WriterJobInput{ChapterID: "chapter-1", PlanRevisionID: "plan-1", Plan: map[string]any{}}
	store.runs["story-1"] = []GenerationRun{{ID: "run-1", StoryID: "story-1", ChapterID: "chapter-1", RunType: "CHAPTER_GENERATION", Status: "PENDING"}}
	store.jobs["run-1"] = []GenerationJob{{
		ID:           "job-1",
		RunID:        "run-1",
		JobType:      "WRITER",
		Status:       "FAILED",
		AttemptCount: 1,
		MaxAttempts:  3,
		LastErrorClass: "TRANSIENT",
	}}
	store.writerInputs["job-1"] = store.currentWriterPlans["chapter-1"]

	handler := NewLatestRunAwareHandler(chiNewRouter(), NewService(store))
	req := httptest.NewRequest(http.MethodGet, "/admin/chapters/chapter-1/content", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got map[string]json.RawMessage
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["generation_job"] == nil || string(got["generation_job"]) == "null" {
		t.Fatalf("expected generation_job projection, got %#v", got)
	}
}

func chiNewRouter() http.Handler {
	return NewHandler(NewService(newFakeStore()), nil)
}
