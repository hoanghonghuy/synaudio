package story_test

import (
	"context"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/story"
)

func TestGetWorkflowSettingsReturnsDefaults(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.workflowSettings["s1"] = story.WorkflowSettings{
		StoryID:        "s1",
		AutoAIReview:   true,
		PauseBeforeTTS: false,
	}

	ws, err := svc.GetWorkflowSettings(context.Background(), "s1")
	if err != nil {
		t.Fatalf("get workflow settings: %v", err)
	}
	if ws.StoryID != "s1" {
		t.Fatalf("expected story s1, got %s", ws.StoryID)
	}
	if !ws.AutoAIReview {
		t.Fatal("expected auto_ai_review true")
	}
}

func TestUpdateWorkflowSettingsPersists(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.workflowSettings["s1"] = story.WorkflowSettings{StoryID: "s1"}

	in := story.WorkflowSettingsInput{
		BatchGenerationSize: 5,
		CreativeAutonomy:    "BALANCED",
		AutoAIReview:        false,
		PauseBeforeTTS:      true,
	}

	ws, err := svc.UpdateWorkflowSettings(context.Background(), "s1", in)
	if err != nil {
		t.Fatalf("update workflow settings: %v", err)
	}
	if ws.BatchGenerationSize != 5 {
		t.Fatalf("expected batch size 5, got %d", ws.BatchGenerationSize)
	}
	if ws.CreativeAutonomy != "BALANCED" {
		t.Fatalf("expected BALANCED, got %q", ws.CreativeAutonomy)
	}
	if ws.AutoAIReview {
		t.Fatal("expected auto_ai_review false")
	}
	if !ws.PauseBeforeTTS {
		t.Fatal("expected pause_before_tts true")
	}
}
