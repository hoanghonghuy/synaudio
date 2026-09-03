package story

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewReadinessHandler exposes read-only Story planning/lifecycle readiness
// contracts separately from mutation endpoints. This keeps the activation
// checklist backend-authoritative for admin clients.
func NewReadinessHandler(svc *Service) http.Handler {
	h := &readinessHandler{svc: svc}
	r := chi.NewRouter()
	r.Get("/admin/stories/{storyID}/activation-readiness", h.getActivationReadiness)
	return r
}

type readinessHandler struct {
	svc *Service
}

func (h *readinessHandler) getActivationReadiness(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.CheckActivationReadiness(r.Context(), chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "STORY_NOT_FOUND", "story not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ready":   result.Ready,
		"missing": result.Missing,
	})
}
