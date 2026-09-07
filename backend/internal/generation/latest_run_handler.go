package generation

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewLatestRunAwareHandler adds the P0 durable chapter-run discovery read route
// in front of the existing generation handler without changing write semantics.
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
	r.Mount("/", base)
	return r
}
