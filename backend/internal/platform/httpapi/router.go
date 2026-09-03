package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/synaudio/synaudio/backend/internal/audit"
)

var ErrDependencyUnavailable = errors.New("dependency unavailable")

type Dependencies struct {
	ReadyCheck            func() error
	DependencyChecks      map[string]func() error
	Logger                *slog.Logger
	AdminCheck            func(context.Context, *http.Request) (bool, error)
	AdminActor            func(context.Context, *http.Request) (string, error)
	AuditRecord           audit.RecordFunc
	AuthHandler           http.Handler
	AuditHandler          http.Handler
	StoryHandler          http.Handler
	StoryReadinessHandler http.Handler
	PlanningHandler       http.Handler
	GenerationHandler     http.Handler
	AudioHandler          http.Handler
	ListenerHandler       http.Handler
	RetconHandler         http.Handler
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
		authHandler := audit.WrapAuth(deps.AuthHandler, deps.AuditRecord, deps.AdminActor)
		r.Mount("/api/v1/auth", authHandler)
	}

	api := chi.NewRouter()
	for _, h := range []http.Handler{
		deps.AuditHandler,
		deps.StoryHandler,
		deps.StoryReadinessHandler,
		deps.PlanningHandler,
		deps.GenerationHandler,
		deps.AudioHandler,
		deps.ListenerHandler,
		deps.RetconHandler,
	} {
		if h == nil {
			continue
		}
		mountRoutes(api, h, deps.AdminCheck, deps.AdminActor, deps.AuditRecord)
	}
	r.Mount("/api/v1", api)

	return r
}

// mountRoutes copies every route registered on src onto dst. Several domain
// handlers share overlapping path prefixes (e.g. /admin/chapters/...), so they
// cannot each be chi.Mount-ed at the same base path; walking their route trees
// and re-registering them on one router avoids the conflict.
func mountRoutes(
	dst chi.Router,
	src http.Handler,
	adminCheck func(context.Context, *http.Request) (bool, error),
	adminActor func(context.Context, *http.Request) (string, error),
	auditRecord audit.RecordFunc,
) {
	routes, ok := src.(chi.Routes)
	if !ok {
		return
	}
	_ = chi.Walk(routes, func(method, route string, handler http.Handler, _ ...func(http.Handler) http.Handler) error {
		if strings.HasPrefix(route, "/admin/") {
			handler = requireAdmin(adminCheck, adminActor)(handler)
		}
		// Audit wraps authorization too, so denied security-sensitive mutations
		// are recorded as DENIED instead of disappearing before the audit layer.
		handler = audit.WrapRoute(handler, method, route, auditRecord, adminActor)
		dst.Method(method, route, handler)
		return nil
	})
}

type adminActorContextKey struct{}

// AdminActorID returns the authenticated admin actor attached by the router.
func AdminActorID(ctx context.Context) string {
	actorID, _ := ctx.Value(adminActorContextKey{}).(string)
	return actorID
}

func withAdminActor(ctx context.Context, actorID string) context.Context {
	return context.WithValue(ctx, adminActorContextKey{}, actorID)
}

func requireAdmin(
	check func(context.Context, *http.Request) (bool, error),
	actor func(context.Context, *http.Request) (string, error),
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if actor != nil {
				actorID, err := actor(r.Context(), r)
				if err != nil || actorID == "" {
					writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
					return
				}
				allowed, err := check(r.Context(), r)
				if err != nil {
					writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
					return
				}
				if !allowed {
					writeError(w, http.StatusForbidden, "FORBIDDEN", "admin access required")
					return
				}
				next.ServeHTTP(w, r.WithContext(withAdminActor(r.Context(), actorID)))
				return
			}
			if check == nil {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
				return
			}
			allowed, err := check(r.Context(), r)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
				return
			}
			if !allowed {
				writeError(w, http.StatusForbidden, "FORBIDDEN", "admin access required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
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
