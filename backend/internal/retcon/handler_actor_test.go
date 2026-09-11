package retcon

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRetconMutationsRejectClientSuppliedActorWithoutAuthenticatedContext(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		store := newFakeStore()
		handler := NewHandler(NewService(store))
		req := httptest.NewRequest(http.MethodPost, "/admin/retcons", strings.NewReader(`{"story_id":"story-1","target_chapter_id":"chapter-1","proposed_change":"change","reason":"reason","requested_by":"spoofed-admin"}`))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
		}
		if got := len(store.retcons["story-1"]); got != 0 {
			t.Fatalf("created retcons = %d, want 0", got)
		}
		if !strings.Contains(rec.Body.String(), "ADMIN_ACTOR_REQUIRED") {
			t.Fatalf("body = %s, want ADMIN_ACTOR_REQUIRED", rec.Body.String())
		}
	})

	t.Run("approve", func(t *testing.T) {
		store := newFakeStore()
		store.retcons["story-1"] = []RetconRequest{{ID: "retcon-1", StoryID: "story-1", Status: "DRAFT"}}
		handler := NewHandler(NewService(store))
		req := httptest.NewRequest(http.MethodPost, "/admin/retcons/retcon-1/approve", strings.NewReader(`{"approved_by":"spoofed-admin"}`))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
		}
		got, err := store.GetRetconRequest(req.Context(), "retcon-1")
		if err != nil {
			t.Fatalf("GetRetconRequest: %v", err)
		}
		if got.Status != "DRAFT" || got.ApprovedBy != "" {
			t.Fatalf("retcon mutated without actor: status=%s approvedBy=%q", got.Status, got.ApprovedBy)
		}
	})

	t.Run("apply", func(t *testing.T) {
		store := newFakeStore()
		store.retcons["story-1"] = []RetconRequest{{ID: "retcon-1", StoryID: "story-1", Status: "READY_TO_APPLY"}}
		handler := NewHandler(NewService(store))
		req := httptest.NewRequest(http.MethodPost, "/admin/retcons/retcon-1/apply", strings.NewReader(`{"applied_by":"spoofed-admin"}`))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
		}
		got, err := store.GetRetconRequest(req.Context(), "retcon-1")
		if err != nil {
			t.Fatalf("GetRetconRequest: %v", err)
		}
		if got.Status != "READY_TO_APPLY" || got.AppliedBy != "" {
			t.Fatalf("retcon mutated without actor: status=%s appliedBy=%q", got.Status, got.AppliedBy)
		}
	})
}
