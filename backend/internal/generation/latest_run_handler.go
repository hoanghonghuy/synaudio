package generation

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewLatestRunAwareHandler adds durable chapter-run discovery in front of the
// existing generation handler without changing write semantics. The content
// projection includes the same authority so existing authenticated frontend
// reads can restore state without introducing a parallel auth transport.
func NewLatestRunAwareHandler(base http.Handler, svc *Service) http.Handler {
	r := chi.NewRouter()
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

	r.Get("/admin/chapters/{chapterID}/content", func(w http.ResponseWriter, req *http.Request) {
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

	r.Mount("/", base)
	return r
}
