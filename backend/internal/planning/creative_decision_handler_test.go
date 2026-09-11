package planning

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreativeDecisionMutationsRejectClientSuppliedActorWithoutAuthenticatedContext(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		store := newFakeStore()
		handler := NewHandler(NewService(store))
		req := httptest.NewRequest(http.MethodPost, "/admin/stories/story-1/creative-decisions", strings.NewReader(`{"decision_type":"PLOT","question":"Should the reveal happen?","created_by":"spoofed-admin"}`))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
		}
		if got := len(store.decisions["story-1"]); got != 0 {
			t.Fatalf("created decisions = %d, want 0", got)
		}
		if !strings.Contains(rec.Body.String(), "ADMIN_ACTOR_REQUIRED") {
			t.Fatalf("body = %s, want ADMIN_ACTOR_REQUIRED", rec.Body.String())
		}
	})

	t.Run("select", func(t *testing.T) {
		store := newFakeStore()
		svc := NewService(store)
		decision, err := svc.CreateCreativeDecision(context.Background(), CreateCreativeDecisionInput{
			StoryID: "story-1", DecisionType: "PLOT", Question: "Q", CreatedBy: "creator",
		})
		if err != nil {
			t.Fatalf("create decision: %v", err)
		}
		handler := NewHandler(svc)
		req := httptest.NewRequest(http.MethodPost, "/admin/creative-decisions/"+decision.ID+"/select", strings.NewReader(`{"selected_by":"spoofed-admin"}`))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
		}
		stored, err := store.GetCreativeDecision(context.Background(), decision.ID)
		if err != nil {
			t.Fatalf("GetCreativeDecision: %v", err)
		}
		if stored.Status != "PROPOSED" || stored.SelectedBy != "" {
			t.Fatalf("decision mutated without actor: status=%s selectedBy=%q", stored.Status, stored.SelectedBy)
		}
	})

	t.Run("reject", func(t *testing.T) {
		store := newFakeStore()
		svc := NewService(store)
		decision, err := svc.CreateCreativeDecision(context.Background(), CreateCreativeDecisionInput{
			StoryID: "story-1", DecisionType: "PLOT", Question: "Q", CreatedBy: "creator",
		})
		if err != nil {
			t.Fatalf("create decision: %v", err)
		}
		handler := NewHandler(svc)
		req := httptest.NewRequest(http.MethodPost, "/admin/creative-decisions/"+decision.ID+"/reject", strings.NewReader(`{"rejected_by":"spoofed-admin","scope":"future-only"}`))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
		}
		stored, err := store.GetCreativeDecision(context.Background(), decision.ID)
		if err != nil {
			t.Fatalf("GetCreativeDecision: %v", err)
		}
		if stored.Status != "PROPOSED" || stored.SelectedBy != "" || stored.RejectionScope != "" {
			t.Fatalf("decision mutated without actor: %#v", stored)
		}
	})
}

func TestWriteCreativeDecisionMutationErrorMapsInvalidTransitionToConflict(t *testing.T) {
	rec := httptest.NewRecorder()

	writeCreativeDecisionMutationError(rec, ErrCreativeDecisionInvalidTransition)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "CREATIVE_DECISION_INVALID_TRANSITION") {
		t.Fatalf("body = %s, want CREATIVE_DECISION_INVALID_TRANSITION", rec.Body.String())
	}
}
