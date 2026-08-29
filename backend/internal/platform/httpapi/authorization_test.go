package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/synaudio/synaudio/backend/internal/platform/httpapi"
)

func TestAdminRoutesRequireAuthentication(t *testing.T) {
	handler := httpapi.NewRouter(httpapi.Dependencies{
		StoryHandler: adminTestHandler(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/secret", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminRoutesRejectNonAdminUsers(t *testing.T) {
	handler := httpapi.NewRouter(httpapi.Dependencies{
		AdminCheck: func(context.Context, *http.Request) (bool, error) {
			return false, nil
		},
		StoryHandler: adminTestHandler(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/secret", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminRoutesAllowAdminUsers(t *testing.T) {
	handler := httpapi.NewRouter(httpapi.Dependencies{
		AdminCheck: func(context.Context, *http.Request) (bool, error) {
			return true, nil
		},
		StoryHandler: adminTestHandler(),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/secret", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPublicRoutesRemainAccessibleWithoutAdmin(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/stories", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := httpapi.NewRouter(httpapi.Dependencies{StoryHandler: r})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stories", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func adminTestHandler() http.Handler {
	r := chi.NewRouter()
	r.Get("/admin/secret", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return r
}
