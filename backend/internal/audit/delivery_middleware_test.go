package audit

import (
	"context"
	"errors"
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

type unavailableAuditStore struct {
	*reliableStore
}

func (s *unavailableAuditStore) CreateAuditEvent(context.Context, Event) (Event, error) {
	return Event{}, errors.New("primary audit persistence unavailable")
}

func (s *unavailableAuditStore) EnqueueAuditIntent(context.Context, Event) error {
	return errors.New("audit outbox persistence unavailable")
}

func TestWrapRouteTransactionalRollsBackBusinessWhenPrimaryAuditPersistenceIsUnavailable(t *testing.T) {
	store := &unavailableAuditStore{reliableStore: newReliableStore()}
	svc := NewService(store)
	businessCommitted := false

	boundary := TransactionBoundary(func(ctx context.Context, run func(context.Context) error) error {
		before := businessCommitted
		if err := run(ctx); err != nil {
			businessCommitted = before
			return err
		}
		return nil
	})

	router := chi.NewRouter()
	router.Method(http.MethodPost, "/admin/stories/{storyID}/archive", WrapRouteTransactional(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			businessCommitted = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"story-1","story_id":"story-1"}`))
		}),
		http.MethodPost,
		"/admin/stories/{storyID}/archive",
		svc.RecordReliable,
		func(context.Context, *http.Request) (string, error) { return "user-1", nil },
		boundary,
	))

	req := httptest.NewRequest(http.MethodPost, "/admin/stories/story-1/archive", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if businessCommitted {
		t.Fatal("business mutation committed without durable audit persistence")
	}
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("rolled-back mutation returned %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	if got := rr.Header().Get("X-Synaudio-Audit-Status"); got != "unavailable" {
		t.Fatalf("expected unavailable audit marker, got %q", got)
	}
	if len(store.finalEvents) != 0 || len(store.intents) != 0 {
		t.Fatalf("unavailable persistence unexpectedly retained audit state: final=%d intents=%d", len(store.finalEvents), len(store.intents))
	}
}
