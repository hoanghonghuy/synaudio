package story

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
	r.Post("/api/v1/admin/stories", h.createStory)
	return r
}

type createStoryRequest struct {
	Title       string                `json:"title"`
	Description string                `json:"description"`
	CreatedBy   string                `json:"created_by"`
	Policy      generationPolicyInput `json:"policy"`
}

type generationPolicyInput struct {
	MinimumAudioDurationSec int    `json:"minimum_audio_duration_sec"`
	TargetAudioDurationSec  int    `json:"target_audio_duration_sec"`
	ContentOrigin           string `json:"content_origin"`
	Language                string `json:"language"`
	NarrationLanguage       string `json:"narration_language"`
}

type storyResponse struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Visibility  string `json:"visibility"`
}

func (h *Handler) createStory(w http.ResponseWriter, r *http.Request) {
	var req createStoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	s, err := h.svc.CreateStory(r.Context(), CreateStoryInput{
		Title:       req.Title,
		Description: req.Description,
		CreatedBy:   req.CreatedBy,
		Policy: GenerationPolicyInput{
			MinimumAudioDurationSec: req.Policy.MinimumAudioDurationSec,
			TargetAudioDurationSec:  req.Policy.TargetAudioDurationSec,
			ContentOrigin:           req.Policy.ContentOrigin,
			Language:                req.Policy.Language,
			NarrationLanguage:       req.Policy.NarrationLanguage,
		},
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidTitle):
			writeError(w, http.StatusBadRequest, "INVALID_TITLE", "invalid title")
		case errors.Is(err, ErrSlugTaken):
			writeError(w, http.StatusConflict, "SLUG_TAKEN", "slug already taken")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, storyResponse{
		ID:          s.ID,
		Slug:        s.Slug,
		Title:       s.Title,
		Description: s.Description,
		Status:      s.Status,
		Visibility:  s.Visibility,
	})
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
