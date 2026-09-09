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
	"github.com/synaudio/synaudio/backend/internal/identity"
)

var ErrDependencyUnavailable = errors.New("dependency unavailable")

type Dependencies struct {
	ReadyCheck               func() error
	DependencyChecks         map[string]func() error
	Logger                   *slog.Logger
	TrustedProxy             TrustedProxyConfig
	AuthAbuse                *AuthAbuse
	AdminCheck               func(context.Context, *http.Request) (bool, error)
	AdminPermissionCheck     func(context.Context, *http.Request, string) (bool, error)
	AdminRecentAuthCheck     func(context.Context, *http.Request) error
	AuthRecentAuthCheck      func(context.Context, *http.Request) error
	AdminActor               func(context.Context, *http.Request) (string, error)
	AuditRecord              audit.RecordFunc
	AuditBoundary            audit.TransactionBoundary
	AuthHandler              http.Handler
	AdminSecurityHandler     http.Handler
	AuditHandler             http.Handler
	StoryHandler             http.Handler
	StoryReadinessHandler    http.Handler
	PlanningHandler          http.Handler
	PlanningWorkspaceHandler http.Handler
	GenerationHandler        http.Handler
	AudioHandler             http.Handler
	ListenerHandler          http.Handler
	RetconHandler            http.Handler
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(WithTrustedClientIP(deps.TrustedProxy))
	r.Use(middleware.Recoverer)
	r.Use(LimitRequestBody(DefaultMaxRequestBodyBytes))
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
		authRouter := chi.NewRouter()
		mountAuthRoutes(authRouter, deps.AuthHandler, deps.AuthRecentAuthCheck)
		authHandler := http.Handler(authRouter)
		if deps.AuthAbuse != nil {
			authHandler = deps.AuthAbuse.Middleware()(authHandler)
		}
		authHandler = audit.WrapAuthTransactional(authHandler, deps.AuditRecord, deps.AdminActor, deps.AuditBoundary)
		r.Mount("/api/v1/auth", authHandler)
	}

	api := chi.NewRouter()
	for _, h := range []http.Handler{
		deps.AdminSecurityHandler,
		deps.AuditHandler,
		deps.StoryHandler,
		deps.StoryReadinessHandler,
		deps.PlanningHandler,
		deps.PlanningWorkspaceHandler,
		deps.GenerationHandler,
		deps.AudioHandler,
		deps.ListenerHandler,
		deps.RetconHandler,
	} {
		if h == nil {
			continue
		}
		mountRoutes(api, h, deps.AdminCheck, deps.AdminPermissionCheck, deps.AdminRecentAuthCheck, deps.AdminActor, deps.AuditRecord, deps.AuditBoundary)
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
	adminPermissionCheck func(context.Context, *http.Request, string) (bool, error),
	adminRecentAuthCheck func(context.Context, *http.Request) error,
	adminActor func(context.Context, *http.Request) (string, error),
	auditRecord audit.RecordFunc,
	auditBoundary audit.TransactionBoundary,
) {
	routes, ok := src.(chi.Routes)
	if !ok {
		return
	}
	_ = chi.Walk(routes, func(method, route string, handler http.Handler, _ ...func(http.Handler) http.Handler) error {
		if strings.HasPrefix(route, "/admin/") {
			policy := adminPolicyFor(method, route)
			if policy.Permission != "" {
				handler = requireAdminPermission(adminPermissionCheck, adminActor, policy.Permission)(handler)
			} else {
				handler = requireAdmin(adminCheck, adminActor)(handler)
			}
			if policy.RecentAuth {
				handler = requireRecentAuth(adminRecentAuthCheck)(handler)
			}
		}
		// Audit wraps authorization too, so denied security-sensitive mutations
		// are recorded as DENIED instead of disappearing before the audit layer.
		handler = audit.WrapRouteTransactional(handler, method, route, auditRecord, adminActor, auditBoundary)
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
				if check == nil {
					writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
					return
				}
				allowed, err := check(r.Context(), r)
				if err != nil {
					writePrivilegedError(w, err)
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
				writePrivilegedError(w, err)
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

func requireAdminPermission(
	check func(context.Context, *http.Request, string) (bool, error),
	actor func(context.Context, *http.Request) (string, error),
	permission string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if actor == nil || check == nil {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
				return
			}
			actorID, err := actor(r.Context(), r)
			if err != nil || actorID == "" {
				writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
				return
			}
			allowed, err := check(r.Context(), r, permission)
			if err != nil {
				writePrivilegedError(w, err)
				return
			}
			if !allowed {
				writeError(w, http.StatusForbidden, "FORBIDDEN", "permission required")
				return
			}
			next.ServeHTTP(w, r.WithContext(withAdminActor(r.Context(), actorID)))
		})
	}
}

// mountAuthRoutes re-registers auth handler routes so high-risk security
// mutations can enforce the same recent-auth boundary used by /admin routes.
func mountAuthRoutes(
	dst chi.Router,
	src http.Handler,
	sessionRecentAuthCheck func(context.Context, *http.Request) error,
) {
	routes, ok := src.(chi.Routes)
	if !ok {
		dst.Mount("/", src)
		return
	}
	_ = chi.Walk(routes, func(method, route string, handler http.Handler, _ ...func(http.Handler) http.Handler) error {
		if authSecurityPolicyFor(method, route).RecentAuth {
			handler = requireRecentAuth(sessionRecentAuthCheck)(handler)
		}
		dst.Method(method, route, handler)
		return nil
	})
}

func requireRecentAuth(check func(context.Context, *http.Request) error) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if check == nil {
				writeError(w, http.StatusForbidden, "RECENT_AUTH_REQUIRED", "recent authentication required")
				return
			}
			if err := check(r.Context(), r); err != nil {
				if errors.Is(err, identity.ErrForbidden) {
					writeError(w, http.StatusForbidden, "RECENT_AUTH_REQUIRED", "recent authentication required")
					return
				}
				writePrivilegedError(w, err)
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

func writePrivilegedError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, identity.ErrMFARequired):
		writeError(w, http.StatusForbidden, "MFA_REQUIRED", "mfa verification required")
	case errors.Is(err, identity.ErrEmailVerificationRequired):
		writeError(w, http.StatusForbidden, "EMAIL_VERIFICATION_REQUIRED", "email verification required")
	case errors.Is(err, identity.ErrForbidden):
		writeError(w, http.StatusForbidden, "FORBIDDEN", "admin access required")
	default:
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
	}
}
