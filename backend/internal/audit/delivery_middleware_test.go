package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestWrapRoutePreservesCommittedResponseAndRetriesAuditDelivery(t *testing.T) {
	store := newReliableStore()
	store.appendFailures = 1
	svc := NewService(store)

	router := chi.NewRouter()
	router.Method(http.MethodPost, "/admin/stories/{storyID}/archive", WrapRoute(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"story-1","story_id":"story-1"}`))
		}),
		http.MethodPost,
		"/admin/stories/{storyID}/archive",
		svc.RecordReliable,
		func(context.Context, *http.Request) (string, error) { return "user-1", nil },
	))

	req := httptest.NewRequest(http.MethodPost, "/admin/stories/story-1/archive", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("business response changed after audit append failure: got %d want %d", rr.Code, http.StatusCreated)
	}
	if got := rr.Header().Get("X-Synaudio-Audit-Status"); got != "" {
		t.Fatalf("durably queued audit should not be reported unavailable, got %q", got)
	}
	if len(store.finalEvents) != 0 || len(store.intents) != 1 {
		t.Fatalf("expected committed business + one queued audit intent, final=%d intents=%d", len(store.finalEvents), len(store.intents))
	}

	report, err := svc.DeliverPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("retry audit delivery: %v", err)
	}
	if report.Delivered != 1 || len(store.finalEvents) != 1 {
		t.Fatalf("expected eventual durable audit event, report=%+v final=%d", report, len(store.finalEvents))
	}
}
