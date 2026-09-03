package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWrapRouteSkipsNonCriticalListenerProgressWrites(t *testing.T) {
	calls := 0
	record := func(_ context.Context, event Event) (Event, error) {
		calls++
		return event, nil
	}
	h := WrapRoute(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), http.MethodPut, "/me/progress/{chapterID}", record, nil)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/me/progress/ch-1", nil))
	if calls != 0 {
		t.Fatalf("expected high-frequency listener progress to stay outside critical audit set, got %d events", calls)
	}
}

func TestEventForResponseNormalizesGoFieldNamesIntoProvenance(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/admin/chapters/ch-1/content", nil)
	body := []byte(`{"ID":"revision-1","ChapterID":"ch-1","PlanRevisionID":"plan-1","GenerationRunID":"run-1"}`)
	event := eventForResponse(req, http.MethodPost, "/admin/chapters/{chapterID}/content", routeDescriptor{
		Action:       "CHAPTER_GENERATED",
		ResourceType: "CONTENT_REVISION",
	}, "user-1", http.StatusCreated, body)

	if event.ResourceID != "revision-1" {
		t.Fatalf("expected content revision resource id, got %q", event.ResourceID)
	}
	if event.GenerationRunID != "run-1" {
		t.Fatalf("expected generation run provenance, got %q", event.GenerationRunID)
	}
	if event.Provenance["plan_revision_id"] != "plan-1" || event.Provenance["generation_run_id"] != "run-1" {
		t.Fatalf("expected normalized provenance refs, got %#v", event.Provenance)
	}
}
