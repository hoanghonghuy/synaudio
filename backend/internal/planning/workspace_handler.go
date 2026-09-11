package planning

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/synaudio/synaudio/backend/internal/platform/httpapi"
)

// NewWorkspaceHandler exposes only version-preserving planning mutations needed
// by Story Planning Studio plus authoritative read projections used by production flows.
// Historical versions are never overwritten.
func NewWorkspaceHandler(svc *Service) http.Handler {
	h := &workspaceHandler{svc: svc}
	r := chi.NewRouter()
	r.Post("/admin/stories/{storyID}/bible/versions", h.createBibleVersion)
	r.Post("/admin/stories/{storyID}/ending/versions", h.createEndingVersion)
	r.Post("/admin/stories/{storyID}/arcs", h.createArc)
	r.Post("/admin/stories/{storyID}/characters", h.createCharacter)
	r.Get("/admin/stories/{storyID}/canon-branches/active-official", h.getActiveOfficialCanonBranch)
	return r
}

type workspaceHandler struct {
	svc *Service
}

type versionContentRequest struct {
	Content map[string]any `json:"content"`
}

func (h *workspaceHandler) createBibleVersion(w http.ResponseWriter, r *http.Request) {
	var req versionContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	created, err := h.svc.CreateBibleVersion(r.Context(), chi.URLParam(r, "storyID"), req.Content, httpapi.AdminActorID(r.Context()))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_BIBLE", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *workspaceHandler) createEndingVersion(w http.ResponseWriter, r *http.Request) {
	var req versionContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	created, err := h.svc.CreateEndingVersion(r.Context(), chi.URLParam(r, "storyID"), req.Content, httpapi.AdminActorID(r.Context()))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ENDING", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

type createArcRequest struct {
	Content map[string]any `json:"content"`
}

func (h *workspaceHandler) createArc(w http.ResponseWriter, r *http.Request) {
	var req createArcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	created, err := h.svc.CreateArc(r.Context(), chi.URLParam(r, "storyID"), req.Content, httpapi.AdminActorID(r.Context()))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARC", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

type createCharacterRequest struct {
	Name       string         `json:"name"`
	Importance string         `json:"importance"`
	Profile    map[string]any `json:"profile"`
}

func (h *workspaceHandler) createCharacter(w http.ResponseWriter, r *http.Request) {
	var req createCharacterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	created, err := h.svc.CreateCharacter(
		r.Context(),
		chi.URLParam(r, "storyID"),
		req.Name,
		req.Importance,
		req.Profile,
		httpapi.AdminActorID(r.Context()),
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CHARACTER", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *workspaceHandler) getActiveOfficialCanonBranch(w http.ResponseWriter, r *http.Request) {
	branch, err := h.svc.GetActiveOfficialCanonBranch(r.Context(), chi.URLParam(r, "storyID"))
	if err != nil {
		if errors.Is(err, ErrCanonBranchNotFound) {
			writeError(w, http.StatusNotFound, "CANON_BRANCH_NOT_FOUND", "active official canon branch not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}
	writeJSON(w, http.StatusOK, branch)
}
