package listener

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc       *Service
	resolveID func(context.Context, *http.Request) (string, error)
}

func NewHandler(svc *Service, resolveID func(context.Context, *http.Request) (string, error)) http.Handler {
	h := &Handler{svc: svc, resolveID: resolveID}

	r := chi.NewRouter()
	r.Get("/me/favorites", h.listFavorites)
	r.Put("/me/favorites/{storyID}", h.addFavorite)
	r.Delete("/me/favorites/{storyID}", h.removeFavorite)
	r.Get("/me/progress/{chapterID}", h.getProgress)
	r.Put("/me/progress/{chapterID}", h.saveProgress)
	r.Post("/me/progress/{chapterID}/complete", h.markCompleted)
	r.Post("/admin/chapters/{chapterID}/revision-impact", h.applyRevisionImpact)
	return r
}

func (h *Handler) listFavorites(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUser(w, r)
	if !ok {
		return
	}
	favs, err := h.svc.ListFavorites(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"favorites": favs})
}

func (h *Handler) addFavorite(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUser(w, r)
	if !ok {
		return
	}
	storyID := chi.URLParam(r, "storyID")

	if err := h.svc.AddFavorite(r.Context(), userID, storyID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "favorited"})
}

func (h *Handler) removeFavorite(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUser(w, r)
	if !ok {
		return
	}
	storyID := chi.URLParam(r, "storyID")

	if err := h.svc.RemoveFavorite(r.Context(), userID, storyID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unfavorited"})
}

func (h *Handler) getProgress(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUser(w, r)
	if !ok {
		return
	}
	chapterID := chi.URLParam(r, "chapterID")

	p, err := h.svc.GetProgress(r.Context(), userID, chapterID)
	if err != nil {
		writeError(w, http.StatusNotFound, "PROGRESS_NOT_FOUND", "progress not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type saveProgressRequest struct {
	PositionMs        int64  `json:"position_ms"`
	AudioAssetID      string `json:"audio_asset_id"`
	PlaybackSessionID string `json:"playback_session_id"`
}

func (h *Handler) saveProgress(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUser(w, r)
	if !ok {
		return
	}
	chapterID := chi.URLParam(r, "chapterID")

	var req saveProgressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	p, err := h.svc.SaveProgress(r.Context(), userID, chapterID, req.PositionMs, req.AudioAssetID, req.PlaybackSessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) markCompleted(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.resolveUser(w, r)
	if !ok {
		return
	}
	chapterID := chi.URLParam(r, "chapterID")

	p, err := h.svc.MarkCompleted(r.Context(), userID, chapterID)
	if err != nil {
		writeError(w, http.StatusNotFound, "PROGRESS_NOT_FOUND", "progress not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) resolveUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	if h.resolveID == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return "", false
	}
	userID, err := h.resolveID(r.Context(), r)
	if err != nil || userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return "", false
	}
	return userID, true
}

type revisionImpactRequest struct {
	RelistenStatus string `json:"relisten_status"`
}

func (h *Handler) applyRevisionImpact(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	var req revisionImpactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	affected, err := h.svc.ApplyRevisionImpact(r.Context(), chapterID, req.RelistenStatus)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"affected_listeners": affected})
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
