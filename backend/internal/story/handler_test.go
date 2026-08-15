package story_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/story"
)

func newTestHandler() http.Handler {
	store := newFakeStore()
	svc := story.NewService(store)
	return story.NewHandler(svc)
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCreateStoryHandlerReturnsCreated(t *testing.T) {
	h := newTestHandler()

	rec := doJSON(t, h, http.MethodPost, "/api/v1/admin/stories", map[string]any{
		"title":       "The Long Road",
		"description": "A journey across the continent.",
		"created_by":  "user-1",
		"policy": map[string]any{
			"minimum_audio_duration_sec": 1200,
			"target_audio_duration_sec":  1800,
			"content_origin":             "ORIGINAL",
			"language":                    "en",
			"narration_language":          "en",
		},
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["id"] == "" {
		t.Fatal("expected id in response")
	}
	if resp["slug"] == "" {
		t.Fatal("expected slug in response")
	}
	if resp["status"] != story.StatusDraft {
		t.Fatalf("expected DRAFT status, got %v", resp["status"])
	}
}

func TestCreateStoryHandlerRejectsEmptyTitle(t *testing.T) {
	h := newTestHandler()

	rec := doJSON(t, h, http.MethodPost, "/api/v1/admin/stories", map[string]any{
		"title":      "   ",
		"created_by": "user-1",
	})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
