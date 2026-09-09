package generation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEditContentHandlerUsesResolvedActor(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	original, err := svc.CreateContentRevision(context.Background(), "chapter-1", "original text", "AI_GENERATED", "writer")
	if err != nil {
		t.Fatalf("create original revision: %v", err)
	}

	handler := NewHandler(svc, func(context.Context, *http.Request) (string, error) {
		return "session-admin", nil
	})
	body := `{"based_on_revision_id":"` + original.ID + `","text":"edited text","edited_by":"attacker"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/chapters/chapter-1/edit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var revision ContentRevision
	if err := json.Unmarshal(rec.Body.Bytes(), &revision); err != nil {
		t.Fatalf("decode revision: %v", err)
	}
	if revision.CreatedBy != "session-admin" {
		t.Fatalf("expected session actor, got %q", revision.CreatedBy)
	}
}

func TestGetGenerationRunEndpointReturnsRunObject(t *testing.T) {
	store := newFakeStore()
	run, err := store.CreateGenerationRun(context.Background(), GenerationRun{
		ID:        "run-1",
		RunType:   "CHAPTER_GENERATION",
		StoryID:   "story-1",
		ChapterID: "chapter-1",
		Status:    "RUNNING",
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	store.jobs[run.ID] = []GenerationJob{{
		ID:      "job-1",
		RunID:   run.ID,
		JobType: "WRITER",
		Status:  "RUNNING",
	}}

	handler := NewHandler(NewService(store), nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/runs/"+run.ID, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var got GenerationRun
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	if got.ID != run.ID || got.Status != run.Status {
		t.Fatalf("expected generation run object, got %#v", got)
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope probe: %v", err)
	}
	if envelope["run"] != nil || envelope["jobs"] != nil {
		t.Fatalf("GET /admin/runs/{runID} must return a GenerationRun object, not an envelope: %#v", envelope)
	}
}

func TestGenerationAdminHandlerRequiresActorResolver(t *testing.T) {
	handler := NewHandler(NewService(newFakeStore()), nil)
	req := httptest.NewRequest(http.MethodPost, "/admin/chapters/chapter-1/edit", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}
