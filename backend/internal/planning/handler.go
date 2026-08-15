package planning

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) http.Handler {
	h := &Handler{svc: svc}

	r := chi.NewRouter()
	r.Post("/admin/stories/{storyID}/foundation", h.generateFoundation)
	r.Get("/admin/stories/{storyID}/bible", h.getBible)
	r.Get("/admin/stories/{storyID}/ending", h.getEnding)
	r.Get("/admin/stories/{storyID}/arcs", h.listArcs)
	r.Get("/admin/stories/{storyID}/characters", h.listCharacters)
	return r
}

type foundationRequest struct {
	Premise   string `json:"premise"`
	CreatedBy string `json:"created_by"`
}

func (h *Handler) generateFoundation(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	var req foundationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	res, err := h.svc.GenerateFoundation(r.Context(), FoundationInput{
		StoryID:   storyID,
		Premise:   req.Premise,
		CreatedBy: req.CreatedBy,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"bible":      res.Bible,
		"ending":     res.Ending,
		"arcs":       res.Arcs,
		"characters": res.Characters,
	})
}

func (h *Handler) getBible(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	bible, err := h.svc.GetCurrentBible(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "BIBLE_NOT_FOUND", "story bible not found")
		return
	}

	writeJSON(w, http.StatusOK, bible)
}

func (h *Handler) getEnding(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	ending, err := h.svc.GetCurrentEnding(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "ENDING_NOT_FOUND", "ending plan not found")
		return
	}

	writeJSON(w, http.StatusOK, ending)
}

func (h *Handler) listArcs(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	arcs, err := h.svc.ListArcs(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"arcs": arcs})
}

func (h *Handler) listCharacters(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	chars, err := h.svc.ListCharacters(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"characters": chars})
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
