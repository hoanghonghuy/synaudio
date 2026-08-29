package story

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
	r.Post("/admin/stories", h.createStory)
	r.Get("/admin/stories", h.listAdminStories)
	r.Get("/stories", h.listPublicStories)
	r.Get("/stories/{storyID}", h.getPublicStory)
	r.Get("/genres", h.listGenres)
	r.Get("/admin/stories/{storyID}/workflow-settings", h.getWorkflowSettings)
	r.Put("/admin/stories/{storyID}/workflow-settings", h.updateWorkflowSettings)
	r.Post("/admin/stories/{storyID}/content-profile", h.createContentProfile)
	r.Get("/admin/stories/{storyID}/content-profile", h.getContentProfile)
	r.Post("/admin/stories/{storyID}/activate", h.activateStory)
	r.Post("/admin/stories/{storyID}/archive", h.archiveStory)
	r.Post("/admin/stories/{storyID}/restore", h.restoreStory)
	r.Post("/admin/stories/{storyID}/make-public", h.makePublic)
	r.Post("/admin/stories/{storyID}/make-private", h.makePrivate)
	r.Post("/admin/stories/{storyID}/cover", h.uploadCover)
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

func (h *Handler) listPublicStories(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	genre := r.URL.Query().Get("genre")
	sort := r.URL.Query().Get("sort")

	if q != "" || genre != "" || sort != "" {
		stories, err := h.svc.SearchStories(r.Context(), SearchStoriesInput{
			Query: q,
			Genre: genre,
			Sort:  sort,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
			return
		}
		writeStoryList(w, stories)
		return
	}

	h.listStories(w, r, true)
}

func (h *Handler) getPublicStory(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.GetPublicStory(r.Context(), chi.URLParam(r, "storyID"))
	if err != nil {
		if errors.Is(err, ErrStoryNotFound) {
			writeError(w, http.StatusNotFound, "STORY_NOT_FOUND", "story not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, storyResponse{
		ID:          st.ID,
		Slug:        st.Slug,
		Title:       st.Title,
		Description: st.Description,
		Status:      st.Status,
		Visibility:  st.Visibility,
	})
}

func writeStoryList(w http.ResponseWriter, stories []Story) {
	out := make([]storyResponse, 0, len(stories))
	for _, s := range stories {
		out = append(out, storyResponse{
			ID:          s.ID,
			Slug:        s.Slug,
			Title:       s.Title,
			Description: s.Description,
			Status:      s.Status,
			Visibility:  s.Visibility,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"stories": out})
}

func (h *Handler) listAdminStories(w http.ResponseWriter, r *http.Request) {
	h.listStories(w, r, false)
}

func (h *Handler) listStories(w http.ResponseWriter, r *http.Request, publicOnly bool) {
	stories, err := h.svc.ListStories(r.Context(), ListStoriesInput{PublicOnly: publicOnly})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	out := make([]storyResponse, 0, len(stories))
	for _, s := range stories {
		out = append(out, storyResponse{
			ID:          s.ID,
			Slug:        s.Slug,
			Title:       s.Title,
			Description: s.Description,
			Status:      s.Status,
			Visibility:  s.Visibility,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"stories": out})
}

func (h *Handler) createStory(w http.ResponseWriter, r *http.Request) {
	var req createStoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	createdBy := req.CreatedBy
	if actorID := httpapi.AdminActorID(r.Context()); actorID != "" {
		createdBy = actorID
	}
	s, err := h.svc.CreateStory(r.Context(), CreateStoryInput{
		Title:       req.Title,
		Description: req.Description,
		CreatedBy:   createdBy,
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

type genreResponse struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type workflowSettingsResponse struct {
	StoryID               string         `json:"story_id"`
	BatchGenerationSize   int            `json:"batch_generation_size"`
	CreativeAutonomy      string         `json:"creative_autonomy"`
	PreferredTextProvider string         `json:"preferred_text_provider"`
	PreferredTextModel    string         `json:"preferred_text_model"`
	PreferredTTSProvider  string         `json:"preferred_tts_provider"`
	PreferredVoiceID      string         `json:"preferred_voice_id"`
	PauseBeforeTTS        bool           `json:"pause_before_tts"`
	AutoAIReview          bool           `json:"auto_ai_review"`
	PlanningHorizon       int            `json:"planning_horizon"`
	FallbackPolicy        map[string]any `json:"fallback_policy"`
}

func (h *Handler) getWorkflowSettings(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	ws, err := h.svc.GetWorkflowSettings(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "STORY_NOT_FOUND", "story not found")
		return
	}

	writeJSON(w, http.StatusOK, toWorkflowSettingsResponse(ws))
}

func (h *Handler) updateWorkflowSettings(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	var req workflowSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	ws, err := h.svc.UpdateWorkflowSettings(r.Context(), storyID, WorkflowSettingsInput{
		BatchGenerationSize:   req.BatchGenerationSize,
		CreativeAutonomy:      req.CreativeAutonomy,
		PreferredTextProvider: req.PreferredTextProvider,
		PreferredTextModel:    req.PreferredTextModel,
		PreferredTTSProvider:  req.PreferredTTSProvider,
		PreferredVoiceID:      req.PreferredVoiceID,
		PauseBeforeTTS:        req.PauseBeforeTTS,
		AutoAIReview:          req.AutoAIReview,
		PlanningHorizon:       req.PlanningHorizon,
		FallbackPolicy:        req.FallbackPolicy,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "STORY_NOT_FOUND", "story not found")
		return
	}

	writeJSON(w, http.StatusOK, toWorkflowSettingsResponse(ws))
}

type workflowSettingsInput struct {
	BatchGenerationSize   int            `json:"batch_generation_size"`
	CreativeAutonomy      string         `json:"creative_autonomy"`
	PreferredTextProvider string         `json:"preferred_text_provider"`
	PreferredTextModel    string         `json:"preferred_text_model"`
	PreferredTTSProvider  string         `json:"preferred_tts_provider"`
	PreferredVoiceID      string         `json:"preferred_voice_id"`
	PauseBeforeTTS        bool           `json:"pause_before_tts"`
	AutoAIReview          bool           `json:"auto_ai_review"`
	PlanningHorizon       int            `json:"planning_horizon"`
	FallbackPolicy        map[string]any `json:"fallback_policy"`
}

func toWorkflowSettingsResponse(ws WorkflowSettings) workflowSettingsResponse {
	return workflowSettingsResponse{
		StoryID:               ws.StoryID,
		BatchGenerationSize:   ws.BatchGenerationSize,
		CreativeAutonomy:      ws.CreativeAutonomy,
		PreferredTextProvider: ws.PreferredTextProvider,
		PreferredTextModel:    ws.PreferredTextModel,
		PreferredTTSProvider:  ws.PreferredTTSProvider,
		PreferredVoiceID:      ws.PreferredVoiceID,
		PauseBeforeTTS:        ws.PauseBeforeTTS,
		AutoAIReview:          ws.AutoAIReview,
		PlanningHorizon:       ws.PlanningHorizon,
		FallbackPolicy:        ws.FallbackPolicy,
	}
}

type contentProfileInput struct {
	MaturityTarget   string         `json:"maturity_target"`
	AllowedThemes    []string       `json:"allowed_themes"`
	DisallowedThemes []string       `json:"disallowed_themes"`
	ViolenceLevel    string         `json:"violence_level"`
	LanguageLimits   string         `json:"language_limits"`
	RomanceLimits    string         `json:"romance_limits"`
	Constraints      map[string]any `json:"constraints"`
}

type contentProfileResponse struct {
	ID        string         `json:"id"`
	StoryID   string         `json:"story_id"`
	VersionNo int            `json:"version_no"`
	Profile   map[string]any `json:"profile"`
}

func (h *Handler) createContentProfile(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	var req contentProfileInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	cp, err := h.svc.CreateContentProfileVersion(r.Context(), storyID, ContentProfileInput{
		MaturityTarget:   req.MaturityTarget,
		AllowedThemes:    req.AllowedThemes,
		DisallowedThemes: req.DisallowedThemes,
		ViolenceLevel:    req.ViolenceLevel,
		LanguageLimits:   req.LanguageLimits,
		RomanceLimits:    req.RomanceLimits,
		Constraints:      req.Constraints,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "STORY_NOT_FOUND", "story not found")
		return
	}

	writeJSON(w, http.StatusCreated, contentProfileResponse{
		ID:        cp.ID,
		StoryID:   cp.StoryID,
		VersionNo: cp.VersionNo,
		Profile:   cp.Profile,
	})
}

func (h *Handler) getContentProfile(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	cp, err := h.svc.GetCurrentContentProfile(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "CONTENT_PROFILE_NOT_FOUND", "content profile not found")
		return
	}

	writeJSON(w, http.StatusOK, contentProfileResponse{
		ID:        cp.ID,
		StoryID:   cp.StoryID,
		VersionNo: cp.VersionNo,
		Profile:   cp.Profile,
	})
}

func (h *Handler) activateStory(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	s, err := h.svc.ActivateStory(r.Context(), storyID)
	if err != nil {
		if errors.Is(err, ErrActivationNotReady) {
			writeError(w, http.StatusConflict, "ACTIVATION_NOT_READY", "activation gate not ready")
			return
		}
		writeError(w, http.StatusNotFound, "STORY_NOT_FOUND", "story not found")
		return
	}

	writeJSON(w, http.StatusOK, storyResponse{
		ID: s.ID, Slug: s.Slug, Title: s.Title, Description: s.Description,
		Status: s.Status, Visibility: s.Visibility,
	})
}

func (h *Handler) archiveStory(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	s, err := h.svc.ArchiveStory(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "STORY_NOT_FOUND", "story not found")
		return
	}

	writeJSON(w, http.StatusOK, storyResponse{
		ID: s.ID, Slug: s.Slug, Title: s.Title, Description: s.Description,
		Status: s.Status, Visibility: s.Visibility,
	})
}

func (h *Handler) restoreStory(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	s, err := h.svc.RestoreStory(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "STORY_NOT_FOUND", "story not found")
		return
	}

	writeJSON(w, http.StatusOK, storyResponse{
		ID: s.ID, Slug: s.Slug, Title: s.Title, Description: s.Description,
		Status: s.Status, Visibility: s.Visibility,
	})
}

func (h *Handler) makePublic(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	s, err := h.svc.MakePublic(r.Context(), storyID)
	if err != nil {
		if errors.Is(err, ErrNotPublicable) {
			writeError(w, http.StatusConflict, "NOT_PUBLICABLE", "story not publicable")
			return
		}
		writeError(w, http.StatusNotFound, "STORY_NOT_FOUND", "story not found")
		return
	}

	writeJSON(w, http.StatusOK, storyResponse{
		ID: s.ID, Slug: s.Slug, Title: s.Title, Description: s.Description,
		Status: s.Status, Visibility: s.Visibility,
	})
}

func (h *Handler) makePrivate(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	s, err := h.svc.MakePrivate(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "STORY_NOT_FOUND", "story not found")
		return
	}

	writeJSON(w, http.StatusOK, storyResponse{
		ID: s.ID, Slug: s.Slug, Title: s.Title, Description: s.Description,
		Status: s.Status, Visibility: s.Visibility,
	})
}

type storyAssetResponse struct {
	ID         string `json:"id"`
	StoryID    string `json:"story_id"`
	Type       string `json:"type"`
	StorageKey string `json:"storage_key"`
	MimeType   string `json:"mime_type"`
	SizeBytes  int64  `json:"size_bytes"`
	Status     string `json:"status"`
}

func (h *Handler) uploadCover(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "missing file")
		return
	}
	defer file.Close()

	data := make([]byte, header.Size)
	if _, err := file.Read(data); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "failed to read file")
		return
	}

	asset, err := h.svc.UploadCover(r.Context(), UploadCoverInput{
		StoryID:     storyID,
		Filename:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Data:        data,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "STORY_NOT_FOUND", "story not found")
		return
	}

	writeJSON(w, http.StatusCreated, storyAssetResponse{
		ID:         asset.ID,
		StoryID:    asset.StoryID,
		Type:       asset.Type,
		StorageKey: asset.StorageKey,
		MimeType:   asset.MimeType,
		SizeBytes:  asset.SizeBytes,
		Status:     asset.Status,
	})
}

func (h *Handler) listGenres(w http.ResponseWriter, r *http.Request) {
	genres, err := h.svc.ListGenres(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	out := make([]genreResponse, 0, len(genres))
	for _, g := range genres {
		out = append(out, genreResponse{ID: g.ID, Slug: g.Slug, Name: g.Name})
	}

	writeJSON(w, http.StatusOK, map[string]any{"genres": out})
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
