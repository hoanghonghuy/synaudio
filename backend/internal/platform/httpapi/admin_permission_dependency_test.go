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
