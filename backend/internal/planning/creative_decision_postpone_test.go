package planning

import (
	"context"
	"errors"
	"testing"
)

func TestPostponeCreativeDecisionPersistsTerminalStateAndProvenance(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	d, err := svc.CreateCreativeDecision(context.Background(), CreateCreativeDecisionInput{
		StoryID:      "s1",
		DecisionType: "PLOT",
		Question:     "Delay the reveal?",
		CreatedBy:    "creator",
	})
	if err != nil {
		t.Fatalf("create decision: %v", err)
	}

	postponed, err := svc.PostponeCreativeDecision(context.Background(), d.ID, "admin-1", "Wait until the next arc")
	if err != nil {
		t.Fatalf("postpone decision: %v", err)
	}
	if postponed.Status != "POSTPONED" {
		t.Fatalf("expected POSTPONED, got %q", postponed.Status)
	}
	if postponed.SelectedBy != "admin-1" {
		t.Fatalf("expected actor provenance admin-1, got %q", postponed.SelectedBy)
	}
	if postponed.RevisitCondition["reason"] != "Wait until the next arc" {
		t.Fatalf("unexpected revisit condition: %#v", postponed.RevisitCondition)
	}

	if _, err := svc.SelectCreativeDecision(context.Background(), d.ID, "admin-2"); !errors.Is(err, ErrCreativeDecisionInvalidTransition) {
		t.Fatalf("expected postponed decision to be terminal, got %v", err)
	}
	stored, err := store.GetCreativeDecision(context.Background(), d.ID)
	if err != nil {
		t.Fatalf("get postponed decision: %v", err)
	}
	if stored.Status != "POSTPONED" || stored.SelectedBy != "admin-1" {
		t.Fatalf("postponed decision mutated after invalid transition: %#v", stored)
	}
}

func TestPostponeCreativeDecisionRejectsMissingReasonWithoutMutation(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	d, _ := svc.CreateCreativeDecision(context.Background(), CreateCreativeDecisionInput{
		StoryID: "s1", DecisionType: "PLOT", Question: "Delay?", CreatedBy: "creator",
	})
	if _, err := svc.PostponeCreativeDecision(context.Background(), d.ID, "admin-1", "   "); err == nil {
		t.Fatal("expected missing postpone reason to fail")
	}
	stored, err := store.GetCreativeDecision(context.Background(), d.ID)
	if err != nil {
		t.Fatalf("get decision: %v", err)
	}
	if stored.Status != "PROPOSED" || stored.SelectedBy != "" || stored.RevisitCondition != nil {
		t.Fatalf("decision mutated after invalid postpone: %#v", stored)
	}
}
