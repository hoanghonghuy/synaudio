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
	r.Get("/chapters/{chapterID}/content", h.getPublishedContent)
	r.Post("/admin/chapters/{chapterID}/approve", h.approveContent)
	r.Post("/admin/chapters/{chapterID}/reviews", h.createReview)
	r.Get("/admin/chapters/{chapterID}/reviews", h.listReviews)
	r.Post("/admin/chapters/{chapterID}/duration", h.analyzeDuration)
	r.Post("/admin/chapters/{chapterID}/continuity", h.runContinuity)
	r.Post("/admin/chapters/{chapterID}/quality", h.runQuality)
	r.Post("/admin/chapters/{chapterID}/safety", h.runSafety)
	r.Post("/admin/chapters/{chapterID}/rewrite", h.rewriteChapter)
	r.Post("/admin/chapters/{chapterID}/edit", h.editContent)
	r.Post("/admin/chapters/{chapterID}/regenerate", h.regenerateContent)
	r.Post("/admin/revisions/{revisionID}/reject", h.rejectContent)
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

func (h *Handler) getPublishedContent(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	revisions, err := h.svc.ListContentRevisions(r.Context(), chapterID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	if len(revisions) == 0 {
		writeError(w, http.StatusNotFound, "CONTENT_NOT_FOUND", "content not found")
		return
	}

	latest := revisions[len(revisions)-1]
	writeJSON(w, http.StatusOK, map[string]any{
		"chapter_id": latest.ChapterID,
		"revision_id": latest.ID,
		"content_text": latest.ContentText,
	})
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

type analyzeDurationRequest struct {
	RevisionID string `json:"revision_id"`
	Text       string `json:"text"`
}

func (h *Handler) analyzeDuration(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	var req analyzeDurationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	out, err := h.svc.AnalyzeDuration(r.Context(), chapterID, req.RevisionID, req.Text)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, out)
}

type runReviewRequest struct {
	RevisionID string `json:"revision_id"`
	Text       string `json:"text"`
}

func (h *Handler) runContinuity(w http.ResponseWriter, r *http.Request) {
	h.runReview(w, r, "CONTINUITY")
}

func (h *Handler) runQuality(w http.ResponseWriter, r *http.Request) {
	h.runReview(w, r, "QUALITY")
}

func (h *Handler) runSafety(w http.ResponseWriter, r *http.Request) {
	h.runReview(w, r, "SAFETY")
}

func (h *Handler) runReview(w http.ResponseWriter, r *http.Request, reviewType string) {
	chapterID := chi.URLParam(r, "chapterID")

	var req runReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	var review ChapterReview
	var err error
	switch reviewType {
	case "CONTINUITY":
		review, err = h.svc.RunContinuityReview(r.Context(), chapterID, req.RevisionID, req.Text)
	case "QUALITY":
		review, err = h.svc.RunQualityReview(r.Context(), chapterID, req.RevisionID, req.Text)
	case "SAFETY":
		review, err = h.svc.RunSafetyReview(r.Context(), chapterID, req.RevisionID, req.Text)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, review)
}

type rewriteRequest struct {
	BasedOnRevisionID string `json:"based_on_revision_id"`
	Feedback          string `json:"feedback"`
	CreatedBy         string `json:"created_by"`
}

func (h *Handler) rewriteChapter(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	var req rewriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	rev, err := h.svc.RewriteChapter(r.Context(), chapterID, req.BasedOnRevisionID, req.Feedback, req.CreatedBy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, rev)
}

type editRequest struct {
	BasedOnRevisionID string `json:"based_on_revision_id"`
	Text              string `json:"text"`
	EditedBy          string `json:"edited_by"`
}

func (h *Handler) editContent(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	var req editRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	rev, err := h.svc.EditContent(r.Context(), chapterID, req.BasedOnRevisionID, req.Text, req.EditedBy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, rev)
}

type regenerateRequest struct {
	BasedOnRevisionID string `json:"based_on_revision_id"`
	RequestedBy       string `json:"requested_by"`
}

func (h *Handler) regenerateContent(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	var req regenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	rev, err := h.svc.RegenerateContent(r.Context(), chapterID, req.BasedOnRevisionID, req.RequestedBy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, rev)
}

type rejectRequest struct {
	RejectedBy string `json:"rejected_by"`
	Reason     string `json:"reason"`
}

func (h *Handler) rejectContent(w http.ResponseWriter, r *http.Request) {
	revisionID := chi.URLParam(r, "revisionID")

	var req rejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	rev, err := h.svc.RejectContent(r.Context(), revisionID, req.RejectedBy, req.Reason)
	if err != nil {
		if errors.Is(err, ErrContentRevisionNotFound) {
			writeError(w, http.StatusNotFound, "CONTENT_REVISION_NOT_FOUND", "content revision not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, rev)
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
