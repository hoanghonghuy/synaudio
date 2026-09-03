package audit

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestWrapRouteRecordsSemanticSuccess(t *testing.T) {
	var got Event
	record := func(_ context.Context, event Event) (Event, error) {
		got = event
		return event, nil
	}
	resolve := func(_ context.Context, _ *http.Request) (string, error) { return "user-1", nil }

	r := chi.NewRouter()
	r.Method(http.MethodPost, "/admin/stories/{storyID}/archive", WrapRoute(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"story-1","status":"ARCHIVED"}`))
		}),
		http.MethodPost,
		"/admin/stories/{storyID}/archive",
		record,
		resolve,
	))

	req := httptest.NewRequest(http.MethodPost, "/admin/stories/story-1/archive", strings.NewReader(`{"password":"must-not-be-read"}`))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got.Action != "STORY_ARCHIVED" || got.ResourceType != "STORY" || got.ResourceID != "story-1" || got.StoryID != "story-1" {
		t.Fatalf("unexpected semantic event: %#v", got)
	}
	if got.ActorUserID != "user-1" || got.ActorType != ActorUser || got.Result != ResultSucceeded {
		t.Fatalf("unexpected actor/result: %#v", got)
	}
	if _, exists := got.Metadata["password"]; exists {
		t.Fatal("request body must never be copied into audit metadata")
	}
}

func TestWrapRouteRecordsDeniedMutation(t *testing.T) {
	var got Event
	record := func(_ context.Context, event Event) (Event, error) {
		got = event
		return event, nil
	}
	resolve := func(_ context.Context, _ *http.Request) (string, error) { return "user-2", nil }

	h := WrapRoute(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}), http.MethodPost, "/admin/stories/{storyID}/archive", record, resolve)

	req := httptest.NewRequest(http.MethodPost, "/admin/stories/story-2/archive", nil)
	ctx := chi.NewRouteContext()
	ctx.URLParams.Add("storyID", "story-2")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden || got.Result != ResultDenied {
		t.Fatalf("expected denied audit result, status=%d event=%#v", rec.Code, got)
	}
}

func TestWrapRouteSurfacesAuditFailureAfterMutation(t *testing.T) {
	mutated := false
	record := func(_ context.Context, _ Event) (Event, error) {
		return Event{}, errors.New("audit unavailable")
	}
	h := WrapRoute(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mutated = true
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"rev-1"}`))
	}), http.MethodPost, "/admin/chapters/{chapterID}/content", record, nil)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/chapters/ch-1/content", nil))
	if !mutated {
		t.Fatal("expected wrapped mutation to execute")
	}
	if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("X-Synaudio-Audit-Status") != "unavailable" {
		t.Fatalf("expected explicit audit failure response, got %d headers=%v", rec.Code, rec.Header())
	}
	if strings.Contains(rec.Body.String(), "rev-1") {
		t.Fatal("must not return original success payload when audit persistence failed")
	}
}

func TestWrapAuthDoesNotAuditReads(t *testing.T) {
	calls := 0
	record := func(_ context.Context, event Event) (Event, error) {
		calls++
		return event, nil
	}
	h := WrapAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), record, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil))
	if calls != 0 {
		t.Fatalf("expected no audit event for read-only auth request, got %d", calls)
	}
}
