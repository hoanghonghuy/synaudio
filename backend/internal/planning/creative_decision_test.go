package planning

import (
	"context"
	"errors"
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

func TestCreativeDecisionTerminalStatusCannotBeRewritten(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	selectedDecision, _ := svc.CreateCreativeDecision(context.Background(), CreateCreativeDecisionInput{
		StoryID: "s1", DecisionType: "PLOT", Question: "Q1", CreatedBy: "creator",
	})
	selected, err := svc.SelectCreativeDecision(context.Background(), selectedDecision.ID, "selector")
	if err != nil {
		t.Fatalf("select decision: %v", err)
	}
	if _, err := svc.RejectCreativeDecision(context.Background(), selected.ID, "attacker", "rewrite"); !errors.Is(err, ErrCreativeDecisionInvalidTransition) {
		t.Fatalf("expected invalid transition selecting->rejecting, got %v", err)
	}
	storedSelected, err := store.GetCreativeDecision(context.Background(), selected.ID)
	if err != nil {
		t.Fatalf("get selected decision: %v", err)
	}
	if storedSelected.Status != "SELECTED" || storedSelected.SelectedBy != "selector" || storedSelected.RejectionScope != "" {
		t.Fatalf("selected decision mutated after rejected transition: %#v", storedSelected)
	}

	rejectedDecision, _ := svc.CreateCreativeDecision(context.Background(), CreateCreativeDecisionInput{
		StoryID: "s1", DecisionType: "PLOT", Question: "Q2", CreatedBy: "creator",
	})
	rejected, err := svc.RejectCreativeDecision(context.Background(), rejectedDecision.ID, "rejector", "future-only")
	if err != nil {
		t.Fatalf("reject decision: %v", err)
	}
	if _, err := svc.SelectCreativeDecision(context.Background(), rejected.ID, "attacker"); !errors.Is(err, ErrCreativeDecisionInvalidTransition) {
		t.Fatalf("expected invalid transition rejecting->selecting, got %v", err)
	}
	storedRejected, err := store.GetCreativeDecision(context.Background(), rejected.ID)
	if err != nil {
		t.Fatalf("get rejected decision: %v", err)
	}
	if storedRejected.Status != "REJECTED" || storedRejected.SelectedBy != "rejector" || storedRejected.RejectionScope != "future-only" {
		t.Fatalf("rejected decision mutated after invalid transition: %#v", storedRejected)
	}
}

func TestCreativeDecisionRepeatedTerminalActionIsRejected(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	d, _ := svc.CreateCreativeDecision(context.Background(), CreateCreativeDecisionInput{
		StoryID: "s1", DecisionType: "PLOT", Question: "Q", CreatedBy: "creator",
	})
	if _, err := svc.SelectCreativeDecision(context.Background(), d.ID, "selector"); err != nil {
		t.Fatalf("select decision: %v", err)
	}
	if _, err := svc.SelectCreativeDecision(context.Background(), d.ID, "different-selector"); !errors.Is(err, ErrCreativeDecisionInvalidTransition) {
		t.Fatalf("expected repeated terminal select to fail, got %v", err)
	}
	stored, err := store.GetCreativeDecision(context.Background(), d.ID)
	if err != nil {
		t.Fatalf("get decision: %v", err)
	}
	if stored.SelectedBy != "selector" {
		t.Fatalf("terminal provenance changed: got %q", stored.SelectedBy)
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
