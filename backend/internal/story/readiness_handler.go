package story

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewReadinessHandler exposes read-only Story planning/lifecycle contracts
// separately from mutation endpoints. This keeps readiness backend-authoritative
// and the immutable policy explicitly read-only for admin clients.
func NewReadinessHandler(svc *Service) http.Handler {
	h := &readinessHandler{svc: svc}
	r := chi.NewRouter()
	r.Get("/admin/stories/{storyID}/activation-readiness", h.getActivationReadiness)
	r.Get("/admin/stories/{storyID}/generation-policy", h.getGenerationPolicy)
	return r
}

type readinessHandler struct {
	svc *Service
}

func generationPolicyResponse(policy GenerationPolicy) map[string]any {
	return map[string]any{
		"story_id":                   policy.StoryID,
		"minimum_audio_duration_sec": policy.MinimumAudioDurationSec,
		"target_audio_duration_sec":  policy.TargetAudioDurationSec,
		"content_origin":             policy.ContentOrigin,
		"language":                   policy.Language,
		"narration_language":         policy.NarrationLanguage,
		"policy_version":             policy.PolicyVersion,
	}
}

func (h *readinessHandler) getActivationReadiness(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	result, err := h.svc.CheckActivationReadiness(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "STORY_NOT_FOUND", "story not found")
		return
	}

	body := map[string]any{
		"ready":   result.Ready,
		"missing": result.Missing,
	}
	if policy, err := h.svc.GetGenerationPolicy(r.Context(), storyID); err == nil {
		body["generation_policy"] = generationPolicyResponse(policy)
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *readinessHandler) getGenerationPolicy(w http.ResponseWriter, r *http.Request) {
	policy, err := h.svc.GetGenerationPolicy(r.Context(), chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusNotFound, "GENERATION_POLICY_NOT_FOUND", "generation policy not found")
		return
	}
	writeJSON(w, http.StatusOK, generationPolicyResponse(policy))
}
