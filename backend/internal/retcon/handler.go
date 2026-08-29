package retcon

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/synaudio/synaudio/backend/internal/platform/httpapi"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) http.Handler {
	h := &Handler{svc: svc}

	r := chi.NewRouter()
	r.Post("/admin/retcons", h.createRetcon)
	r.Get("/admin/retcons", h.listRetcons)
	r.Get("/admin/retcons/{id}", h.getRetcon)
	r.Post("/admin/retcons/{id}/approve", h.approveRetcon)
	r.Post("/admin/retcons/{id}/cancel", h.cancelRetcon)
	r.Post("/admin/retcons/{id}/analyze", h.analyzeRetcon)
	r.Post("/admin/retcons/{id}/ready", h.markReadyRetcon)
	r.Post("/admin/retcons/{id}/apply", h.applyRetcon)
	return r
}

type createRetconRequest struct {
	StoryID         string `json:"story_id"`
	TargetChapterID string `json:"target_chapter_id"`
	ProposedChange  string `json:"proposed_change"`
	Reason          string `json:"reason"`
	RequestedBy     string `json:"requested_by"`
}

func (h *Handler) createRetcon(w http.ResponseWriter, r *http.Request) {
	var req createRetconRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	requestedBy := req.RequestedBy
	if actorID := httpapi.AdminActorID(r.Context()); actorID != "" {
		requestedBy = actorID
	}
	ret, err := h.svc.CreateRetconRequest(r.Context(), CreateRetconInput{
		StoryID:         req.StoryID,
		TargetChapterID: req.TargetChapterID,
		ProposedChange:  req.ProposedChange,
		Reason:          req.Reason,
		RequestedBy:     requestedBy,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, ret)
}

func (h *Handler) listRetcons(w http.ResponseWriter, r *http.Request) {
	storyID := r.URL.Query().Get("story_id")
	if storyID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "story_id is required")
		return
	}

	list, err := h.svc.ListRetconRequests(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"retcons": list})
}

func (h *Handler) getRetcon(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	ret, err := h.svc.store.GetRetconRequest(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrRetconNotFound) {
			writeError(w, http.StatusNotFound, "RETCON_NOT_FOUND", "retcon not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, ret)
}

type approveRetconRequest struct {
	ApprovedBy string `json:"approved_by"`
}

func (h *Handler) approveRetcon(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req approveRetconRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	approvedBy := req.ApprovedBy
	if actorID := httpapi.AdminActorID(r.Context()); actorID != "" {
		approvedBy = actorID
	}
	ret, err := h.svc.ApproveRetconRequest(r.Context(), id, approvedBy)
	if err != nil {
		if errors.Is(err, ErrRetconNotFound) {
			writeError(w, http.StatusNotFound, "RETCON_NOT_FOUND", "retcon not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, ret)
}

func (h *Handler) cancelRetcon(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	ret, err := h.svc.CancelRetconRequest(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrRetconNotFound) {
			writeError(w, http.StatusNotFound, "RETCON_NOT_FOUND", "retcon not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, ret)
}

func (h *Handler) analyzeRetcon(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	ret, err := h.svc.AnalyzeRetconRequest(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrRetconNotFound) {
			writeError(w, http.StatusNotFound, "RETCON_NOT_FOUND", "retcon not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, ret)
}

func (h *Handler) markReadyRetcon(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	ret, err := h.svc.MarkReadyToApply(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrRetconNotReady) {
			writeError(w, http.StatusConflict, "RETCON_NOT_READY", "retcon not approved")
			return
		}
		if errors.Is(err, ErrRetconNotFound) {
			writeError(w, http.StatusNotFound, "RETCON_NOT_FOUND", "retcon not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, ret)
}

type applyRetconRequest struct {
	AppliedBy string `json:"applied_by"`
}

func (h *Handler) applyRetcon(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req applyRetconRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	appliedBy := req.AppliedBy
	if actorID := httpapi.AdminActorID(r.Context()); actorID != "" {
		appliedBy = actorID
	}
	ret, err := h.svc.ApplyRetconRequest(r.Context(), id, appliedBy)
	if err != nil {
		if errors.Is(err, ErrRetconNotReady) {
			writeError(w, http.StatusConflict, "RETCON_NOT_READY", "retcon not ready to apply")
			return
		}
		if errors.Is(err, ErrRetconNotFound) {
			writeError(w, http.StatusNotFound, "RETCON_NOT_FOUND", "retcon not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, ret)
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
