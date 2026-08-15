package generation

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) http.Handler {
	h := &Handler{svc: svc}

	r := chi.NewRouter()
	r.Post("/admin/chapters/{chapterID}/content", h.writeChapter)
	r.Get("/admin/chapters/{chapterID}/content", h.listContentRevisions)
	r.Post("/admin/chapters/{chapterID}/approve", h.approveContent)
	r.Post("/admin/chapters/{chapterID}/reviews", h.createReview)
	r.Get("/admin/chapters/{chapterID}/reviews", h.listReviews)
	r.Post("/admin/runs", h.createRun)
	r.Get("/admin/runs/{runID}", h.getRun)
	r.Post("/admin/runs/{runID}/jobs", h.createJob)
	return r
}

type writeChapterRequest struct {
	Prompt    string `json:"prompt"`
	CreatedBy string `json:"created_by"`
}

func (h *Handler) writeChapter(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	var req writeChapterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	rev, err := h.svc.WriteChapter(r.Context(), chapterID, req.Prompt, req.CreatedBy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, rev)
}

func (h *Handler) listContentRevisions(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	revisions, err := h.svc.ListContentRevisions(r.Context(), chapterID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"revisions": revisions})
}

type approveContentRequest struct {
	RevisionID string `json:"revision_id"`
	ApprovedBy string `json:"approved_by"`
}

func (h *Handler) approveContent(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	var req approveContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	a, err := h.svc.ApproveContent(r.Context(), chapterID, req.RevisionID, req.ApprovedBy)
	if err != nil {
		if errors.Is(err, ErrContentRevisionNotFound) {
			writeError(w, http.StatusNotFound, "CONTENT_REVISION_NOT_FOUND", "content revision not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, a)
}

type createReviewRequest struct {
	ContentRevisionID string         `json:"content_revision_id"`
	ReviewType        string         `json:"review_type"`
	Outcome           string         `json:"outcome"`
	Report            map[string]any `json:"report"`
}

func (h *Handler) createReview(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	var req createReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	review, err := h.svc.CreateChapterReview(r.Context(), chapterID, req.ContentRevisionID, req.ReviewType, req.Outcome, req.Report)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REVIEW", "invalid review")
		return
	}

	writeJSON(w, http.StatusCreated, review)
}

func (h *Handler) listReviews(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	reviews, err := h.svc.ListChapterReviews(r.Context(), chapterID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"reviews": reviews})
}

type createRunRequest struct {
	RunType     string `json:"run_type"`
	StoryID     string `json:"story_id"`
	ChapterID   string `json:"chapter_id"`
	RequestedBy string `json:"requested_by"`
}

func (h *Handler) createRun(w http.ResponseWriter, r *http.Request) {
	var req createRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	run, err := h.svc.CreateGenerationRun(r.Context(), req.RunType, req.StoryID, req.ChapterID, req.RequestedBy)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_RUN", "invalid run")
		return
	}

	writeJSON(w, http.StatusCreated, run)
}

func (h *Handler) getRun(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")

	run, err := h.svc.GetGenerationRun(r.Context(), runID)
	if err != nil {
		if errors.Is(err, ErrGenerationRunNotFound) {
			writeError(w, http.StatusNotFound, "RUN_NOT_FOUND", "run not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, run)
}

type createJobRequest struct {
	JobType     string `json:"job_type"`
	MaxAttempts int    `json:"max_attempts"`
}

func (h *Handler) createJob(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")

	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	job, err := h.svc.CreateGenerationJob(r.Context(), runID, req.JobType, req.MaxAttempts)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JOB", "invalid job")
		return
	}

	writeJSON(w, http.StatusCreated, job)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
