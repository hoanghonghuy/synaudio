package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/synaudio/synaudio/backend/internal/audit"
)

func TestRouterAuditsRepresentativeCriticalAdminMutations(t *testing.T) {
	src := chi.NewRouter()
	src.Post("/admin/stories/{storyID}/archive", okJSONHandler)
	src.Post("/admin/chapters/{chapterID}/approve", okJSONHandler)
	src.Post("/admin/retcons/{id}/apply", okJSONHandler)
	src.Post("/admin/chapters/{chapterID}/audio/{assetID}/activate", okJSONHandler)
	src.Post("/admin/canon-branches/{branchID}/commit", okJSONHandler)

	actorID := "11111111-1111-1111-1111-111111111111"
	var events []audit.Event
	router := NewRouter(Dependencies{
		AdminPermissionCheck: func(context.Context, *http.Request, string) (bool, error) { return true, nil },
		AdminActor:           func(context.Context, *http.Request) (string, error) { return actorID, nil },
		AuditRecord: func(_ context.Context, event audit.Event) (audit.Event, error) {
			events = append(events, event)
			return event, nil
		},
		StoryHandler: src,
	})

	cases := []struct {
		path   string
		action string
	}{
		{"/api/v1/admin/stories/22222222-2222-2222-2222-222222222222/archive", "STORY_ARCHIVED"},
		{"/api/v1/admin/chapters/33333333-3333-3333-3333-333333333333/approve", "CHAPTER_APPROVED"},
		{"/api/v1/admin/retcons/44444444-4444-4444-4444-444444444444/apply", "RETCON_APPLIED"},
		{"/api/v1/admin/chapters/33333333-3333-3333-3333-333333333333/audio/55555555-5555-5555-5555-555555555555/activate", "AUDIO_ACTIVATED"},
		{"/api/v1/admin/canon-branches/66666666-6666-6666-6666-666666666666/commit", "CANON_COMMITTED"},
	}

	for _, tc := range cases {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tc.path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d: %s", tc.path, rec.Code, rec.Body.String())
		}
		got := events[len(events)-1]
		if got.Action != tc.action || got.ActorUserID != actorID || got.Result != audit.ResultSucceeded {
			t.Fatalf("%s: unexpected audit event %#v", tc.path, got)
		}
	}
}

func TestRouterAuditsDeniedAdminMutation(t *testing.T) {
	src := chi.NewRouter()
	src.Post("/admin/stories/{storyID}/archive", okJSONHandler)
	actorID := "11111111-1111-1111-1111-111111111111"
	var got audit.Event
	router := NewRouter(Dependencies{
		AdminPermissionCheck: func(context.Context, *http.Request, string) (bool, error) { return false, nil },
		AdminActor:           func(context.Context, *http.Request) (string, error) { return actorID, nil },
		AuditRecord: func(_ context.Context, event audit.Event) (audit.Event, error) {
			got = event
			return event, nil
		},
		StoryHandler: src,
	})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/admin/stories/22222222-2222-2222-2222-222222222222/archive", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
	if got.Action != "STORY_ARCHIVED" || got.Result != audit.ResultDenied || got.ActorUserID != actorID {
		t.Fatalf("unexpected denied audit event: %#v", got)
	}
}

func TestRouterAuditsAuthSecurityMutationWithoutRequestBodyInspection(t *testing.T) {
	authHandler := chi.NewRouter()
	authHandler.Post("/logout", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"logged_out"}`))
	})
	actorID := "11111111-1111-1111-1111-111111111111"
	var got audit.Event
	router := NewRouter(Dependencies{
		AdminActor: func(context.Context, *http.Request) (string, error) { return actorID, nil },
		AuditRecord: func(_ context.Context, event audit.Event) (audit.Event, error) {
			got = event
			return event, nil
		},
		AuthHandler: authHandler,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got.Action != "AUTH_LOGOUT" || got.ActorUserID != actorID || got.Result != audit.ResultSucceeded {
		t.Fatalf("unexpected auth audit event: %#v", got)
	}
}

func okJSONHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
