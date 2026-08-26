package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var ErrDependencyUnavailable = errors.New("dependency unavailable")

type Dependencies struct {
	ReadyCheck       func() error
	DependencyChecks map[string]func() error
	Logger           *slog.Logger
	AuthHandler      http.Handler
	StoryHandler     http.Handler
	PlanningHandler  http.Handler
	GenerationHandler http.Handler
	AudioHandler     http.Handler
	ListenerHandler  http.Handler
	RetconHandler    http.Handler
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	if deps.Logger != nil {
		r.Use(WithRequestLogger(deps.Logger))
	}

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Get("/ready", func(w http.ResponseWriter, _ *http.Request) {
		status := http.StatusOK
		body := map[string]any{"status": "ready"}

		if deps.ReadyCheck != nil {
			if err := deps.ReadyCheck(); err != nil {
				status = http.StatusServiceUnavailable
				body["status"] = "unavailable"
				body["error"] = "dependency_unavailable"
			}
		}

		if len(deps.DependencyChecks) > 0 {
			depStatus := map[string]string{}
			allOK := true
			for name, check := range deps.DependencyChecks {
				if check == nil {
					depStatus[name] = "ok"
					continue
				}
				if err := check(); err != nil {
					depStatus[name] = "unavailable"
					allOK = false
				} else {
					depStatus[name] = "ok"
				}
			}
			body["dependencies"] = depStatus
			if !allOK {
				status = http.StatusServiceUnavailable
				body["status"] = "degraded"
			}
		}

		writeJSON(w, status, body)
	})

	if deps.AuthHandler != nil {
		r.Mount("/api/v1/auth", deps.AuthHandler)
	}

	api := chi.NewRouter()
	for _, h := range []http.Handler{
		deps.StoryHandler,
		deps.PlanningHandler,
		deps.GenerationHandler,
		deps.AudioHandler,
		deps.ListenerHandler,
		deps.RetconHandler,
	} {
		if h == nil {
			continue
		}
		mountRoutes(api, h)
	}
	r.Mount("/api/v1", api)

	return r
}

// mountRoutes copies every route registered on src onto dst. Several domain
// handlers share overlapping path prefixes (e.g. /admin/chapters/...), so they
// cannot each be chi.Mount-ed at the same base path; walking their route trees
// and re-registering them on one router avoids the conflict.
func mountRoutes(dst chi.Router, src http.Handler) {
	routes, ok := src.(chi.Routes)
	if !ok {
		return
	}
	_ = chi.Walk(routes, func(method, route string, handler http.Handler, _ ...func(http.Handler) http.Handler) error {
		dst.Method(method, route, handler)
		return nil
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
