package audit

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) http.Handler {
	h := &Handler{svc: svc}
	r := chi.NewRouter()
	r.Get("/admin/audit", h.list)
	r.Get("/admin/audit/{eventID}", h.get)
	return r
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := Filter{
		ActorUserID:     q.Get("actor_id"),
		Action:          q.Get("action"),
		ResourceType:    q.Get("resource_type"),
		ResourceID:      q.Get("resource_id"),
		StoryID:         q.Get("story_id"),
		ChapterID:       q.Get("chapter_id"),
		GenerationRunID: q.Get("run_id"),
		CorrelationID:   q.Get("correlation_id"),
		Result:          q.Get("result"),
	}
	if raw := q.Get("from"); raw != "" {
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_FROM", "from must be RFC3339")
			return
		}
		filter.From = value
	}
	if raw := q.Get("to"); raw != "" {
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_TO", "to must be RFC3339")
			return
		}
		filter.To = value
	}
	if raw := q.Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			writeError(w, http.StatusBadRequest, "INVALID_LIMIT", "limit must be a positive integer")
			return
		}
		filter.Limit = value
	}

	events, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": events})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	event, err := h.svc.Get(r.Context(), chi.URLParam(r, "eventID"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "AUDIT_NOT_FOUND", "audit event not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
