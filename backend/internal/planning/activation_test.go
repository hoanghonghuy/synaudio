package planning

import (
	"context"
	"testing"
)

func TestCheckActivationReadyAllPresent(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	store.bibles["s1"] = []StoryBibleVersion{{ID: "b1", StoryID: "s1"}}
	store.endings["s1"] = []EndingPlanVersion{{ID: "e1", StoryID: "s1"}}
	store.arcs["s1"] = []StoryArc{{ID: "a1", StoryID: "s1"}}
	store.characters["s1"] = []Character{{ID: "c1", StoryID: "s1"}}

	missing, err := svc.CheckActivationReady(context.Background(), "s1")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("expected no missing, got %v", missing)
	}
}

func TestCheckActivationReadyMissingBible(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	store.endings["s1"] = []EndingPlanVersion{{ID: "e1", StoryID: "s1"}}
	store.arcs["s1"] = []StoryArc{{ID: "a1", StoryID: "s1"}}
	store.characters["s1"] = []Character{{ID: "c1", StoryID: "s1"}}

	missing, err := svc.CheckActivationReady(context.Background(), "s1")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(missing) != 1 || missing[0] != "story_bible" {
		t.Fatalf("expected [story_bible], got %v", missing)
	}
}

func TestCheckActivationReadyMissingArcs(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	store.bibles["s1"] = []StoryBibleVersion{{ID: "b1", StoryID: "s1"}}
	store.endings["s1"] = []EndingPlanVersion{{ID: "e1", StoryID: "s1"}}
	store.characters["s1"] = []Character{{ID: "c1", StoryID: "s1"}}

	missing, err := svc.CheckActivationReady(context.Background(), "s1")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(missing) != 1 || missing[0] != "arc" {
		t.Fatalf("expected [arc], got %v", missing)
	}
}

func TestCheckActivationReadyMissingCharacters(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	store.bibles["s1"] = []StoryBibleVersion{{ID: "b1", StoryID: "s1"}}
	store.endings["s1"] = []EndingPlanVersion{{ID: "e1", StoryID: "s1"}}
	store.arcs["s1"] = []StoryArc{{ID: "a1", StoryID: "s1"}}

	missing, err := svc.CheckActivationReady(context.Background(), "s1")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(missing) != 1 || missing[0] != "character" {
		t.Fatalf("expected [character], got %v", missing)
	}
}
