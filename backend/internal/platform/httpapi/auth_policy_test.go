package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func TestAuthMFADisableRequiresRecentAuthPolicy(t *testing.T) {
	if !authSecurityPolicyFor(http.MethodPost, "/mfa/totp/disable").RecentAuth {
		t.Fatal("MFA disable must require recent-auth")
	}
	if authSecurityPolicyFor(http.MethodPost, "/mfa/totp/setup").RecentAuth {
		t.Fatal("MFA setup must not require recent-auth")
	}
	if authSecurityPolicyFor(http.MethodPost, "/re-auth").RecentAuth {
		t.Fatal("re-auth must not require recent-auth")
	}
}

func TestAuthHighRiskRouteRejectsStaleRecentAuth(t *testing.T) {
	cases := []struct {
		name  string
		path  string
		route string
	}{
		{name: "mfa disable", path: "/api/v1/auth/mfa/totp/disable", route: "/mfa/totp/disable"},
		{name: "account deletion", path: "/api/v1/auth/account/deletion/request", route: "/account/deletion/request"},
	}
	for _, tc := range cases {
		src := chi.NewRouter()
		executed := false
		src.Post(tc.route, func(w http.ResponseWriter, _ *http.Request) {
			executed = true
			okJSONHandler(w, nil)
		})

		router := NewRouter(Dependencies{
			AuthRecentAuthCheck: func(context.Context, *http.Request) error { return identity.ErrForbidden },
			AdminActor:          func(context.Context, *http.Request) (string, error) { return "user-1", nil },
			AuthHandler:         src,
		})

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tc.path, nil))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s: expected 403, got %d: %s", tc.name, rec.Code, rec.Body.String())
		}
		if !contains(rec.Body.String(), "RECENT_AUTH_REQUIRED") {
			t.Fatalf("%s: expected RECENT_AUTH_REQUIRED, got %s", tc.name, rec.Body.String())
		}
		if executed {
			t.Fatalf("%s: handler must not execute without fresh recent-auth", tc.name)
		}
	}
}

func TestAuthHighRiskRouteAllowsFreshRecentAuth(t *testing.T) {
	src := chi.NewRouter()
	src.Post("/mfa/totp/disable", okJSONHandler)

	router := NewRouter(Dependencies{
		AuthRecentAuthCheck: func(context.Context, *http.Request) error { return nil },
		AdminActor:          func(context.Context, *http.Request) (string, error) { return "user-1", nil },
		AuthHandler:         src,
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/totp/disable", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthHighRiskRouteRejectsUnauthenticatedBeforeHandler(t *testing.T) {
	src := chi.NewRouter()
	executed := false
	src.Post("/account/deletion/request", func(w http.ResponseWriter, _ *http.Request) {
		executed = true
		okJSONHandler(w, nil)
	})

	router := NewRouter(Dependencies{
		AuthRecentAuthCheck: func(context.Context, *http.Request) error { return identity.ErrUnauthenticated },
		AuthHandler:         src,
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/account/deletion/request", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
	if executed {
		t.Fatal("unauthenticated request must not reach handler")
	}
}

func TestAuthHighRiskRouteFailsClosedWhenRecentAuthCheckerMissing(t *testing.T) {
	src := chi.NewRouter()
	executed := false
	src.Post("/mfa/totp/disable", func(w http.ResponseWriter, _ *http.Request) {
		executed = true
		okJSONHandler(w, nil)
	})

	router := NewRouter(Dependencies{
		AuthRecentAuthCheck: nil,
		AuthHandler:         src,
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/totp/disable", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
	if executed {
		t.Fatal("handler must not execute when recent-auth checker is missing")
	}
}

func TestAuthNonHighRiskRouteDoesNotRequireRecentAuth(t *testing.T) {
	src := chi.NewRouter()
	src.Post("/mfa/totp/setup", okJSONHandler)
	called := false

	router := NewRouter(Dependencies{
		AuthRecentAuthCheck: func(context.Context, *http.Request) error {
			called = true
			return errors.New("stale")
		},
		AuthHandler: src,
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/totp/setup", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if called {
		t.Fatal("non-high-risk auth route must not invoke recent-auth checker")
	}
}
