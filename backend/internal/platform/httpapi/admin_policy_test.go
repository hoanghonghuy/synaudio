package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestAdminRouteUsesOperationSpecificPermission(t *testing.T) {
	src := chi.NewRouter()
	src.Post("/admin/retcons/{id}/apply", okJSONHandler)

	var gotPermission string
	router := NewRouter(Dependencies{
		AdminPermissionCheck: func(_ context.Context, _ *http.Request, permission string) (bool, error) {
			gotPermission = permission
			return false, nil
		},
		AdminRecentAuthCheck: func(context.Context, *http.Request) error { return nil },
		AdminActor: func(context.Context, *http.Request) (string, error) { return "actor", nil },
		RetconHandler: src,
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/retcons/retcon-1/apply", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected permission denial 403, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotPermission != "RETCON_APPLY" {
		t.Fatalf("expected RETCON_APPLY permission check, got %q", gotPermission)
	}
}

func TestHighRiskAdminRouteRejectsStaleRecentAuth(t *testing.T) {
	src := chi.NewRouter()
	src.Post("/admin/retcons/{id}/apply", okJSONHandler)

	router := NewRouter(Dependencies{
		AdminPermissionCheck: func(context.Context, *http.Request, string) (bool, error) { return true, nil },
		AdminRecentAuthCheck: func(context.Context, *http.Request) error { return errors.New("stale") },
		AdminActor: func(context.Context, *http.Request) (string, error) { return "actor", nil },
		RetconHandler: src,
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/retcons/retcon-1/apply", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected recent-auth denial 403, got %d: %s", rec.Code, rec.Body.String())
	}
	if !contains(rec.Body.String(), "RECENT_AUTH_REQUIRED") {
		t.Fatalf("expected RECENT_AUTH_REQUIRED error, got %s", rec.Body.String())
	}
}

func TestNonHighRiskAdminRouteDoesNotRequireRecentAuth(t *testing.T) {
	src := chi.NewRouter()
	src.Post("/admin/stories", okJSONHandler)
	calledRecent := false
	router := NewRouter(Dependencies{
		AdminPermissionCheck: func(_ context.Context, _ *http.Request, permission string) (bool, error) {
			return permission == "STORY_CREATE", nil
		},
		AdminRecentAuthCheck: func(context.Context, *http.Request) error {
			calledRecent = true
			return errors.New("stale")
		},
		AdminActor: func(context.Context, *http.Request) (string, error) { return "actor", nil },
		StoryHandler: src,
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/stories", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if calledRecent {
		t.Fatal("ordinary story creation must not require recent auth")
	}
}

func contains(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle { return true }
	}
	return false
}
