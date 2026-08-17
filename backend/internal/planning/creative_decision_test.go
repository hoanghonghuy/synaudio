package planning

import (
	"context"
	"testing"
)

func TestCreateCreativeDecisionProposes(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	d, err := svc.CreateCreativeDecision(context.Background(), CreateCreativeDecisionInput{
		StoryID:      "s1",
		ChapterID:    "ch1",
		DecisionType: "PLOT",
		Severity:     "MAJOR",
		Question:     "Should Lan die?",
		CreatedBy:    "u1",
	})
	if err != nil {
		t.Fatalf("create decision: %v", err)
	}
	if d.Status != "PROPOSED" {
		t.Fatalf("expected PROPOSED, got %q", d.Status)
	}
	if d.Severity != "MAJOR" {
		t.Fatalf("expected MAJOR, got %q", d.Severity)
	}
}

func TestCreateCreativeDecisionRejectsEmptyQuestion(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.CreateCreativeDecision(context.Background(), CreateCreativeDecisionInput{
		StoryID: "s1",
	}); err == nil {
		t.Fatal("expected error for empty question")
	}
}

func TestSelectCreativeDecisionSetsSelected(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	d, _ := svc.CreateCreativeDecision(context.Background(), CreateCreativeDecisionInput{
		StoryID:      "s1",
		DecisionType: "PLOT",
		Severity:     "MAJOR",
		Question:     "Should Lan die?",
		CreatedBy:    "u1",
	})

	selected, err := svc.SelectCreativeDecision(context.Background(), d.ID, "u1")
	if err != nil {
		t.Fatalf("select decision: %v", err)
	}
	if selected.Status != "SELECTED" {
		t.Fatalf("expected SELECTED, got %q", selected.Status)
	}
	if selected.SelectedBy != "u1" {
		t.Fatalf("expected selected_by u1, got %q", selected.SelectedBy)
	}
}

func TestRejectCreativeDecision(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	d, _ := svc.CreateCreativeDecision(context.Background(), CreateCreativeDecisionInput{
		StoryID:      "s1",
		DecisionType: "PLOT",
		Severity:     "MAJOR",
		Question:     "Should Lan die?",
		CreatedBy:    "u1",
	})

	rejected, err := svc.RejectCreativeDecision(context.Background(), d.ID, "u1", "not now")
	if err != nil {
		t.Fatalf("reject decision: %v", err)
	}
	if rejected.Status != "REJECTED" {
		t.Fatalf("expected REJECTED, got %q", rejected.Status)
	}
}

func TestListCreativeDecisionsByStory(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	_, _ = svc.CreateCreativeDecision(context.Background(), CreateCreativeDecisionInput{
		StoryID: "s1", DecisionType: "PLOT", Severity: "MAJOR", Question: "Q1", CreatedBy: "u1",
	})
	_, _ = svc.CreateCreativeDecision(context.Background(), CreateCreativeDecisionInput{
		StoryID: "s1", DecisionType: "PLOT", Severity: "MINOR", Question: "Q2", CreatedBy: "u1",
	})

	list, err := svc.ListCreativeDecisions(context.Background(), "s1")
	if err != nil {
		t.Fatalf("list decisions: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 decisions, got %d", len(list))
	}
}
