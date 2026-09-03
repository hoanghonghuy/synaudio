package story

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewReadinessHandler exposes Story workspace contracts that were previously
// missing from the thin admin surface. Domain mutations remain narrow and
// preserve immutable policy/version/lifecycle fields.
func NewReadinessHandler(svc *Service) http.Handler {
	h := &readinessHandler{svc: svc}
	r := chi.NewRouter()
	r.Get("/admin/stories/{storyID}/activation-readiness", h.getActivationReadiness)
	r.Get("/admin/stories/{storyID}/generation-policy", h.getGenerationPolicy)
	r.Put("/admin/stories/{storyID}/metadata", h.updateMetadata)
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
	if details, err := h.svc.GetWorkspaceDetails(r.Context(), storyID); err == nil {
		body["story_workspace"] = map[string]any{
			"planning_mode":   details.PlanningMode,
			"planning_phase":  details.PlanningPhase,
			"public_rating":   details.PublicRating,
			"public_warnings": details.PublicWarnings,
			"cover_asset_id":  details.CoverAssetID,
		}
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

type updateMetadataRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (h *readinessHandler) updateMetadata(w http.ResponseWriter, r *http.Request) {
	var req updateMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	st, err := h.svc.UpdateMetadata(r.Context(), chi.URLParam(r, "storyID"), req.Title, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidTitle):
			writeError(w, http.StatusBadRequest, "INVALID_TITLE", "invalid title")
		case errors.Is(err, ErrStoryNotFound):
			writeError(w, http.StatusNotFound, "STORY_NOT_FOUND", "story not found")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":          st.ID,
		"slug":        st.Slug,
		"title":       st.Title,
		"description": st.Description,
		"status":      st.Status,
		"visibility":  st.Visibility,
	})
}
