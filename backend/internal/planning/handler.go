package planning

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
	r.Post("/admin/stories/{storyID}/foundation", h.generateFoundation)
	r.Get("/admin/stories/{storyID}/bible", h.getBible)
	r.Get("/admin/stories/{storyID}/ending", h.getEnding)
	r.Get("/admin/stories/{storyID}/arcs", h.listArcs)
	r.Get("/admin/stories/{storyID}/characters", h.listCharacters)
	r.Post("/admin/stories/{storyID}/chapters", h.createChapter)
	r.Get("/admin/stories/{storyID}/chapters", h.listChapters)
	r.Get("/stories/{storyID}/chapters", h.listPublishedChapters)
	r.Post("/admin/chapters/{chapterID}/plans", h.createPlanRevision)
	r.Post("/admin/stories/{storyID}/facts", h.createFact)
	r.Get("/admin/stories/{storyID}/facts", h.listFacts)
	r.Post("/admin/stories/{storyID}/plot-threads", h.createPlotThread)
	r.Get("/admin/stories/{storyID}/plot-threads", h.listPlotThreads)
	r.Post("/admin/plot-threads/{threadID}/events", h.createPlotThreadEvent)
	r.Post("/admin/stories/{storyID}/canon-branches", h.createCanonBranch)
	r.Post("/admin/canon-branches/{branchID}/versions", h.createCanonVersion)
	r.Get("/admin/canon-branches/{branchID}/versions", h.listCanonVersions)
	r.Post("/admin/canon-branches/{branchID}/provisional-versions", h.createProvisionalCanonVersion)
	r.Post("/admin/canon-versions/{versionID}/promote", h.promoteProvisionalVersion)
	r.Post("/admin/stories/{storyID}/context-snapshots", h.createContextSnapshot)
	r.Get("/admin/stories/{storyID}/context-snapshots", h.listContextSnapshots)
	r.Post("/admin/canon-branches/{branchID}/commit", h.commitCanon)
	r.Post("/admin/chapters/{chapterID}/publish", h.publishChapter)
	r.Post("/admin/chapters/{chapterID}/unpublish", h.unpublishChapter)
	r.Get("/admin/stories/{storyID}/creative-decisions", h.listCreativeDecisions)
	r.Post("/admin/stories/{storyID}/creative-decisions", h.createCreativeDecision)
	r.Post("/admin/creative-decisions/{decisionID}/select", h.selectCreativeDecision)
	r.Post("/admin/creative-decisions/{decisionID}/reject", h.rejectCreativeDecision)
	r.Get("/admin/stories/{storyID}/arcs/{arcID}/completion", h.reviewArcCompletion)
	r.Get("/admin/stories/{storyID}/attention", h.listAttentionItems)
	r.Post("/admin/stories/{storyID}/attention", h.createAttentionItem)
	r.Post("/admin/attention/{itemID}/resolve", h.resolveAttentionItem)
	r.Get("/admin/context-snapshots/{snapshotID}", h.getContextSnapshot)
	r.Get("/admin/stories/{storyID}/thread-inactivity", h.analyzeThreadInactivity)
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

type createChapterRequest struct {
	Title     string `json:"title"`
	CreatedBy string `json:"created_by"`
}

func (h *Handler) createChapter(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	var req createChapterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	c, err := h.svc.CreateChapter(r.Context(), storyID, req.Title, req.CreatedBy)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_TITLE", "invalid title")
		return
	}

	writeJSON(w, http.StatusCreated, c)
}

func (h *Handler) listChapters(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	chapters, err := h.svc.ListChapters(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"chapters": chapters})
}

func (h *Handler) listPublishedChapters(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	chapters, err := h.svc.ListChapters(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	published := make([]Chapter, 0, len(chapters))
	for _, c := range chapters {
		if c.Status == "PUBLISHED" {
			published = append(published, c)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"chapters": published})
}

type createPlanRequest struct {
	Plan      map[string]any `json:"plan"`
	CreatedBy string         `json:"created_by"`
}

func (h *Handler) createPlanRevision(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	var req createPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	p, err := h.svc.CreatePlanRevision(r.Context(), chapterID, req.Plan, req.CreatedBy)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_PLAN", "invalid plan")
		return
	}

	writeJSON(w, http.StatusCreated, p)
}

type createFactRequest struct {
	SubjectType string         `json:"subject_type"`
	SubjectID   string         `json:"subject_id"`
	FactType    string         `json:"fact_type"`
	Value       map[string]any `json:"value"`
	Importance  string         `json:"importance"`
}

func (h *Handler) createFact(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	var req createFactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	f, err := h.svc.CreateFact(r.Context(), storyID, req.SubjectType, req.SubjectID, req.FactType, req.Value, req.Importance)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_FACT", "invalid fact")
		return
	}

	writeJSON(w, http.StatusCreated, f)
}

func (h *Handler) listFacts(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	facts, err := h.svc.ListFacts(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"facts": facts})
}

type createPlotThreadRequest struct {
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Importance string `json:"importance"`
}

func (h *Handler) createPlotThread(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	var req createPlotThreadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	t, err := h.svc.CreatePlotThread(r.Context(), storyID, req.Title, req.Summary, req.Importance)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_THREAD", "invalid plot thread")
		return
	}

	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) listPlotThreads(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	threads, err := h.svc.ListPlotThreads(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"plot_threads": threads})
}

type createPlotThreadEventRequest struct {
	EventType string         `json:"event_type"`
	ChapterID string         `json:"chapter_id"`
	Detail    map[string]any `json:"detail"`
}

func (h *Handler) createPlotThreadEvent(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "threadID")

	var req createPlotThreadEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	e, err := h.svc.CreatePlotThreadEvent(r.Context(), threadID, req.EventType, req.ChapterID, req.Detail)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_EVENT", "invalid event")
		return
	}

	writeJSON(w, http.StatusCreated, e)
}

type createCanonBranchRequest struct {
	Type string `json:"type"`
}

func (h *Handler) createCanonBranch(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	var req createCanonBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	b, err := h.svc.CreateCanonBranch(r.Context(), storyID, req.Type)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, b)
}

type createCanonVersionRequest struct {
	StoryID         string `json:"story_id"`
	SourceChapterID string `json:"source_chapter_id"`
	CommittedBy     string `json:"committed_by"`
}

func (h *Handler) createCanonVersion(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchID")

	var req createCanonVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	v, err := h.svc.CreateCanonVersion(r.Context(), req.StoryID, branchID, req.SourceChapterID, req.CommittedBy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, v)
}

func (h *Handler) listCanonVersions(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchID")

	versions, err := h.svc.ListCanonVersions(r.Context(), branchID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"versions": versions})
}

type createProvisionalCanonVersionRequest struct {
	StoryID              string `json:"story_id"`
	SourceChapterID      string `json:"source_chapter_id"`
	SourceContentRevisionID string `json:"source_content_revision_id"`
}

func (h *Handler) createProvisionalCanonVersion(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchID")

	var req createProvisionalCanonVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	v, err := h.svc.CreateProvisionalCanonVersion(r.Context(), req.StoryID, branchID, req.SourceChapterID, req.SourceContentRevisionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, v)
}

type promoteProvisionalVersionRequest struct {
	CommittedBy string `json:"committed_by"`
}

func (h *Handler) promoteProvisionalVersion(w http.ResponseWriter, r *http.Request) {
	versionID := chi.URLParam(r, "versionID")

	var req promoteProvisionalVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	v, err := h.svc.PromoteProvisionalVersion(r.Context(), versionID, req.CommittedBy)
	if err != nil {
		if errors.Is(err, ErrCanonVersionNotFound) {
			writeError(w, http.StatusNotFound, "VERSION_NOT_FOUND", "version not found")
			return
		}
		writeError(w, http.StatusConflict, "PROMOTE_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, v)
}

type createContextSnapshotRequest struct {
	ChapterID               string `json:"chapter_id"`
	BibleVersionID          string `json:"bible_version_id"`
	EndingPlanVersionID     string `json:"ending_plan_version_id"`
	ArcVersionID            string `json:"arc_version_id"`
	ContentProfileVersionID string `json:"content_profile_version_id"`
	PromptVersion           string `json:"prompt_version"`
	WorkflowVersion         string `json:"workflow_version"`
	Provider                string `json:"provider"`
	Model                   string `json:"model"`
}

func (h *Handler) createContextSnapshot(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	var req createContextSnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	sn, err := h.svc.CreateContextSnapshot(r.Context(), storyID, req.ChapterID, req.BibleVersionID, req.EndingPlanVersionID, req.ArcVersionID, req.ContentProfileVersionID, req.PromptVersion, req.WorkflowVersion, req.Provider, req.Model)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, sn)
}

func (h *Handler) listContextSnapshots(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	snapshots, err := h.svc.ListContextSnapshots(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"snapshots": snapshots})
}

type commitCanonRequest struct {
	StoryID           string `json:"story_id"`
	SourceChapterID   string `json:"source_chapter_id"`
	ContentRevisionID string `json:"content_revision_id"`
	CommittedBy       string `json:"committed_by"`
}

func (h *Handler) commitCanon(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchID")

	var req commitCanonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	res, err := h.svc.CommitCanon(r.Context(), req.StoryID, branchID, req.SourceChapterID, req.ContentRevisionID, req.CommittedBy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, res)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (h *Handler) publishChapter(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	ch, err := h.svc.PublishChapter(r.Context(), chapterID)
	if err != nil {
		if errors.Is(err, ErrPublishNotReady) {
			writeError(w, http.StatusConflict, "PUBLISH_NOT_READY", "chapter not ready to publish")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, ch)
}

func (h *Handler) unpublishChapter(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	ch, err := h.svc.UnpublishChapter(r.Context(), chapterID)
	if err != nil {
		if errors.Is(err, ErrNotPublished) {
			writeError(w, http.StatusConflict, "NOT_PUBLISHED", "chapter not published")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, ch)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

type createCreativeDecisionRequest struct {
	ChapterID      string `json:"chapter_id"`
	ArcID          string `json:"arc_id"`
	Origin         string `json:"origin"`
	DecisionType   string `json:"decision_type"`
	Severity       string `json:"severity"`
	BlockingLevel  string `json:"blocking_level"`
	Question       string `json:"question"`
	ContextSummary string `json:"context_summary"`
	CreatedBy      string `json:"created_by"`
}

func (h *Handler) createCreativeDecision(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	var req createCreativeDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	d, err := h.svc.CreateCreativeDecision(r.Context(), CreateCreativeDecisionInput{
		StoryID:        storyID,
		ChapterID:      req.ChapterID,
		ArcID:          req.ArcID,
		Origin:         req.Origin,
		DecisionType:   req.DecisionType,
		Severity:       req.Severity,
		BlockingLevel:  req.BlockingLevel,
		Question:       req.Question,
		ContextSummary: req.ContextSummary,
		CreatedBy:      req.CreatedBy,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, d)
}

func (h *Handler) listCreativeDecisions(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	decisions, err := h.svc.ListCreativeDecisions(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"decisions": decisions})
}

type selectCreativeDecisionRequest struct {
	SelectedBy string `json:"selected_by"`
}

func (h *Handler) selectCreativeDecision(w http.ResponseWriter, r *http.Request) {
	decisionID := chi.URLParam(r, "decisionID")

	var req selectCreativeDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	d, err := h.svc.SelectCreativeDecision(r.Context(), decisionID, req.SelectedBy)
	if err != nil {
		if errors.Is(err, ErrCreativeDecisionNotFound) {
			writeError(w, http.StatusNotFound, "DECISION_NOT_FOUND", "decision not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, d)
}

type rejectCreativeDecisionRequest struct {
	RejectedBy string `json:"rejected_by"`
	Scope      string `json:"scope"`
}

func (h *Handler) rejectCreativeDecision(w http.ResponseWriter, r *http.Request) {
	decisionID := chi.URLParam(r, "decisionID")

	var req rejectCreativeDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	d, err := h.svc.RejectCreativeDecision(r.Context(), decisionID, req.RejectedBy, req.Scope)
	if err != nil {
		if errors.Is(err, ErrCreativeDecisionNotFound) {
			writeError(w, http.StatusNotFound, "DECISION_NOT_FOUND", "decision not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, d)
}

func (h *Handler) reviewArcCompletion(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	arcID := chi.URLParam(r, "arcID")

	res, err := h.svc.ReviewArcCompletion(r.Context(), storyID, arcID)
	if err != nil {
		if errors.Is(err, ErrArcNotFound) {
			writeError(w, http.StatusNotFound, "ARC_NOT_FOUND", "arc not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, res)
}

type createAttentionItemRequest struct {
	ChapterID string `json:"chapter_id"`
	Priority  string `json:"priority"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	Action    string `json:"action"`
}

func (h *Handler) createAttentionItem(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	var req createAttentionItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	item, err := h.svc.CreateAttentionItem(r.Context(), AttentionItemInput{
		StoryID:   storyID,
		ChapterID: req.ChapterID,
		Priority:  req.Priority,
		Kind:      req.Kind,
		Title:     req.Title,
		Detail:    req.Detail,
		Action:    req.Action,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) listAttentionItems(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	items, err := h.svc.ListAttentionItems(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) resolveAttentionItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")

	item, err := h.svc.ResolveAttentionItem(r.Context(), itemID)
	if err != nil {
		if errors.Is(err, ErrAttentionItemNotFound) {
			writeError(w, http.StatusNotFound, "ATTENTION_NOT_FOUND", "attention item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) getContextSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshotID := chi.URLParam(r, "snapshotID")

	sn, err := h.svc.GetContextSnapshot(r.Context(), snapshotID)
	if err != nil {
		if errors.Is(err, ErrContextSnapshotNotFound) {
			writeError(w, http.StatusNotFound, "SNAPSHOT_NOT_FOUND", "snapshot not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, sn)
}

func (h *Handler) analyzeThreadInactivity(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	res, err := h.svc.AnalyzeThreadInactivity(r.Context(), storyID, 1)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"inactive_threads": res})
}
