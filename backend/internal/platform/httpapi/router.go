package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var ErrDependencyUnavailable = errors.New("dependency unavailable")

type Dependencies struct {
	ReadyCheck   func() error
	AuthHandler  http.Handler
	StoryHandler http.Handler
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Get("/ready", func(w http.ResponseWriter, _ *http.Request) {
		if deps.ReadyCheck != nil {
			if err := deps.ReadyCheck(); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{
					"status": "unavailable",
					"error":  "dependency_unavailable",
				})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	if deps.AuthHandler != nil {
		r.Mount("/api/v1/auth", deps.AuthHandler)
	}
	if deps.StoryHandler != nil {
		r.Mount("/api/v1", deps.StoryHandler)
	}

	return r
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
