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

func TestUnmappedAdminRouteFailsClosedInsteadOfFallingBackToBroadAdmin(t *testing.T) {
	src := chi.NewRouter()
	src.Post("/admin/future-sensitive-action", okJSONHandler)

	var gotPermission string
	broadAdminCalled := false
	router := NewRouter(Dependencies{
		AdminCheck: func(context.Context, *http.Request) (bool, error) {
			broadAdminCalled = true
			return true, nil
		},
		AdminPermissionCheck: func(_ context.Context, _ *http.Request, permission string) (bool, error) {
			gotPermission = permission
			return false, nil
		},
		AdminActor: func(context.Context, *http.Request) (string, error) { return "actor", nil },
		StoryHandler: src,
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/future-sensitive-action", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected unmapped privileged route to fail closed with 403, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotPermission != unmappedAdminPermission {
		t.Fatalf("expected fail-closed sentinel permission, got %q", gotPermission)
	}
	if broadAdminCalled {
		t.Fatal("unmapped privileged route must not fall back to broad admin authorization")
	}
}

func TestStoryLifecycleRoutesUseSpecificPermissions(t *testing.T) {
	cases := []struct {
		route string
		want  string
	}{
		{"/admin/stories/{storyID}/activate", "STORY_ACTIVATE"},
		{"/admin/stories/{storyID}/archive", "STORY_ARCHIVE"},
		{"/admin/stories/{storyID}/restore", "STORY_RESTORE"},
		{"/admin/stories/{storyID}/make-public", "STORY_VISIBILITY_MANAGE"},
		{"/admin/stories/{storyID}/make-private", "STORY_VISIBILITY_MANAGE"},
	}
	for _, tc := range cases {
		if got := adminPolicyFor(http.MethodPost, tc.route).Permission; got != tc.want {
			t.Fatalf("%s: expected %s, got %s", tc.route, tc.want, got)
		}
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
