package generation

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

const adminChapterContentRoute = "/admin/chapters/{chapterID}/content"

// NewLatestRunAwareHandler composes the existing generation routes with durable
// chapter-run discovery. It replaces, rather than mounts over, the existing
// content read route so httpapi's route flattening cannot silently restore the
// older revisions-only projection.
func NewLatestRunAwareHandler(base http.Handler, svc *Service) http.Handler {
	baseRoutes, ok := base.(chi.Routes)
	if !ok {
		panic("generation base handler must expose chi routes")
	}

	r := chi.NewRouter()
	if err := chi.Walk(baseRoutes, func(method, route string, handler http.Handler, _ ...func(http.Handler) http.Handler) error {
		if method == http.MethodGet && route == adminChapterContentRoute {
			return nil
		}
		r.Method(method, route, handler)
		return nil
	}); err != nil {
		panic("generation base routes could not be composed: " + err.Error())
	}

	r.Get("/admin/chapters/{chapterID}/generation-run/latest", func(w http.ResponseWriter, req *http.Request) {
		chapterID := chi.URLParam(req, "chapterID")
		run, err := svc.GetLatestChapterGenerationRun(req.Context(), chapterID)
		if err != nil {
			if errors.Is(err, ErrGenerationRunNotFound) {
				writeError(w, http.StatusNotFound, "RUN_NOT_FOUND", "chapter generation run not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
			return
		}
		writeJSON(w, http.StatusOK, run)
	})

	r.Get(adminChapterContentRoute, func(w http.ResponseWriter, req *http.Request) {
		chapterID := chi.URLParam(req, "chapterID")
		revisions, err := svc.ListContentRevisions(req.Context(), chapterID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
			return
		}

		var run *GenerationRun
		latest, err := svc.GetLatestChapterGenerationRun(req.Context(), chapterID)
		switch {
		case err == nil:
			run = &latest
		case errors.Is(err, ErrGenerationRunNotFound):
			// Explicit empty authority; do not create or infer a run.
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"revisions":      revisions,
			"generation_run": run,
		})
	})

	return r
}
