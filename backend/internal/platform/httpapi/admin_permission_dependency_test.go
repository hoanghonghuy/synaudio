package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestMappedAdminRouteFailsClosedWhenPermissionCheckerMissing(t *testing.T) {
	src := chi.NewRouter()
	src.Post("/admin/stories", okJSONHandler)

	broadAdminCalled := false
	router := NewRouter(Dependencies{
		AdminCheck: func(context.Context, *http.Request) (bool, error) {
			broadAdminCalled = true
			return true, nil
		},
		AdminPermissionCheck: nil,
		AdminActor:           func(context.Context, *http.Request) (string, error) { return "actor", nil },
		StoryHandler:         src,
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/stories", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected mapped privileged route to fail closed with 401, got %d: %s", rec.Code, rec.Body.String())
	}
	if broadAdminCalled {
		t.Fatal("mapped privileged route must not fall back to broad admin authorization when permission checker is missing")
	}
}

func TestUnmappedAdminRouteFailsClosedWhenPermissionCheckerMissing(t *testing.T) {
	src := chi.NewRouter()
	src.Post("/admin/future-sensitive-action", okJSONHandler)

	broadAdminCalled := false
	router := NewRouter(Dependencies{
		AdminCheck: func(context.Context, *http.Request) (bool, error) {
			broadAdminCalled = true
			return true, nil
		},
		AdminPermissionCheck: nil,
		AdminActor:           func(context.Context, *http.Request) (string, error) { return "actor", nil },
		StoryHandler:         src,
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/future-sensitive-action", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unmapped privileged route to fail closed with 401, got %d: %s", rec.Code, rec.Body.String())
	}
	if broadAdminCalled {
		t.Fatal("unmapped privileged route must not fall back to broad admin authorization when permission checker is missing")
	}
}

func TestHighRiskAdminRouteFailsClosedWhenRecentAuthCheckerMissing(t *testing.T) {
	src := chi.NewRouter()
	executed := false
	src.Post("/admin/users/{userID}/roles/admin", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		executed = true
		w.WriteHeader(http.StatusNoContent)
	}))

	permissionCalled := false
	router := NewRouter(Dependencies{
		AdminPermissionCheck: func(_ context.Context, _ *http.Request, permission string) (bool, error) {
			permissionCalled = true
			if permission != "ADMIN_ROLE_GRANT" {
				t.Fatalf("expected ADMIN_ROLE_GRANT permission, got %q", permission)
			}
			return true, nil
		},
		AdminRecentAuthCheck: nil,
		AdminActor:           func(context.Context, *http.Request) (string, error) { return "actor", nil },
		AdminSecurityHandler: src,
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/target/roles/admin", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected high-risk privileged route to fail closed with 403, got %d: %s", rec.Code, rec.Body.String())
	}
	if executed {
		t.Fatal("high-risk privileged handler must not execute when recent-auth checker is missing")
	}
	if permissionCalled {
		t.Fatal("recent-auth middleware should reject the request before the permission-protected handler chain executes")
	}
}
