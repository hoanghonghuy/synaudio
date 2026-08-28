package story_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/synaudio/synaudio/backend/internal/story"
)

func newTestHandler() http.Handler {
	store := newFakeStore()
	svc := story.NewService(store)
	return wrapHandler(svc)
}

func wrapHandler(svc *story.Service) http.Handler {
	r := chi.NewRouter()
	r.Mount("/api/v1", story.NewHandler(svc))
	return r
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

func TestListGenresHandlerReturnsGenres(t *testing.T) {
	store := newFakeStore()
	store.genres = []story.Genre{
		{ID: "g1", Slug: "fantasy", Name: "Fantasy"},
		{ID: "g2", Slug: "romance", Name: "Romance"},
	}
	svc := story.NewService(store)
	h := wrapHandler(svc)

	rec := doJSON(t, h, http.MethodGet, "/api/v1/genres", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	genres, ok := resp["genres"].([]any)
	if !ok {
		t.Fatalf("expected genres array, got %v", resp["genres"])
	}
	if len(genres) != 2 {
		t.Fatalf("expected 2 genres, got %d", len(genres))
	}
}

func TestListStoriesPublicHandlerReturnsPublicOnly(t *testing.T) {
	store := newFakeStore()
	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Visibility: story.VisibilityPublic}
	store.stories["s2"] = story.Story{ID: "s2", Slug: "b", Title: "B", Visibility: story.VisibilityPrivate}
	svc := story.NewService(store)
	h := wrapHandler(svc)

	rec := doJSON(t, h, http.MethodGet, "/api/v1/stories", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	stories, ok := resp["stories"].([]any)
	if !ok {
		t.Fatalf("expected stories array, got %v", resp["stories"])
	}
	if len(stories) != 1 {
		t.Fatalf("expected 1 public story, got %d", len(stories))
	}
}

func TestGetPublicStoryHandlerReturnsStory(t *testing.T) {
	store := newFakeStore()
	store.stories["s1"] = story.Story{
		ID:         "s1",
		Slug:       "the-long-road",
		Title:      "The Long Road",
		Description: "A journey across the continent.",
		Status:     story.StatusActive,
		Visibility: story.VisibilityPublic,
	}
	svc := story.NewService(store)
	h := wrapHandler(svc)

	rec := doJSON(t, h, http.MethodGet, "/api/v1/stories/s1", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["id"] != "s1" {
		t.Fatalf("expected story s1, got %v", resp["id"])
	}
	if resp["title"] != "The Long Road" {
		t.Fatalf("expected story title, got %v", resp["title"])
	}
}

func TestGetPrivateStoryHandlerReturnsNotFound(t *testing.T) {
	store := newFakeStore()
	store.stories["s1"] = story.Story{
		ID:         "s1",
		Title:      "Private Draft",
		Visibility: story.VisibilityPrivate,
	}
	svc := story.NewService(store)
	h := wrapHandler(svc)

	rec := doJSON(t, h, http.MethodGet, "/api/v1/stories/s1", nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListStoriesAdminHandlerReturnsAll(t *testing.T) {
	store := newFakeStore()
	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Visibility: story.VisibilityPublic}
	store.stories["s2"] = story.Story{ID: "s2", Slug: "b", Title: "B", Visibility: story.VisibilityPrivate}
	svc := story.NewService(store)
	h := wrapHandler(svc)

	rec := doJSON(t, h, http.MethodGet, "/api/v1/admin/stories", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	stories, ok := resp["stories"].([]any)
	if !ok {
		t.Fatalf("expected stories array, got %v", resp["stories"])
	}
	if len(stories) != 2 {
		t.Fatalf("expected 2 stories, got %d", len(stories))
	}
}

func TestGetWorkflowSettingsHandlerReturnsSettings(t *testing.T) {
	store := newFakeStore()
	store.workflowSettings["s1"] = story.WorkflowSettings{StoryID: "s1", AutoAIReview: true}
	svc := story.NewService(store)
	h := wrapHandler(svc)

	rec := doJSON(t, h, http.MethodGet, "/api/v1/admin/stories/s1/workflow-settings", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["auto_ai_review"] != true {
		t.Fatalf("expected auto_ai_review true, got %v", resp["auto_ai_review"])
	}
}

func TestUpdateWorkflowSettingsHandlerPersists(t *testing.T) {
	store := newFakeStore()
	store.workflowSettings["s1"] = story.WorkflowSettings{StoryID: "s1"}
	svc := story.NewService(store)
	h := wrapHandler(svc)

	rec := doJSON(t, h, http.MethodPut, "/api/v1/admin/stories/s1/workflow-settings", map[string]any{
		"batch_generation_size": 5,
		"creative_autonomy":     "BALANCED",
		"auto_ai_review":        false,
		"pause_before_tts":      true,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["batch_generation_size"] != float64(5) {
		t.Fatalf("expected batch size 5, got %v", resp["batch_generation_size"])
	}
}

func TestCreateContentProfileHandlerReturnsCreated(t *testing.T) {
	store := newFakeStore()
	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A"}
	svc := story.NewService(store)
	h := wrapHandler(svc)

	rec := doJSON(t, h, http.MethodPost, "/api/v1/admin/stories/s1/content-profile", map[string]any{
		"maturity_target": "TEEN",
		"allowed_themes":  []string{"adventure"},
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["version_no"] != float64(1) {
		t.Fatalf("expected version 1, got %v", resp["version_no"])
	}
}

func TestGetContentProfileHandlerReturnsCurrent(t *testing.T) {
	store := newFakeStore()
	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A"}
	svc := story.NewService(store)
	h := wrapHandler(svc)

	doJSON(t, h, http.MethodPost, "/api/v1/admin/stories/s1/content-profile", map[string]any{
		"maturity_target": "TEEN",
	})

	rec := doJSON(t, h, http.MethodGet, "/api/v1/admin/stories/s1/content-profile", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	profile, ok := resp["profile"].(map[string]any)
	if !ok {
		t.Fatalf("expected profile object, got %v", resp["profile"])
	}
	if profile["maturity_target"] != "TEEN" {
		t.Fatalf("expected TEEN, got %v", profile["maturity_target"])
	}
}

func TestArchiveStoryHandlerReturnsArchived(t *testing.T) {
	store := newFakeStore()
	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Status: story.StatusActive}
	svc := story.NewService(store)
	h := wrapHandler(svc)

	rec := doJSON(t, h, http.MethodPost, "/api/v1/admin/stories/s1/archive", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["status"] != story.StatusArchived {
		t.Fatalf("expected ARCHIVED, got %v", resp["status"])
	}
}

func TestMakePublicHandlerRejectsDraft(t *testing.T) {
	store := newFakeStore()
	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Status: story.StatusDraft}
	svc := story.NewService(store)
	h := wrapHandler(svc)

	rec := doJSON(t, h, http.MethodPost, "/api/v1/admin/stories/s1/make-public", nil)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUploadCoverHandlerReturnsCreated(t *testing.T) {
	store := newFakeStore()
	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A"}
	storage := newFakeStorage()
	svc := story.NewService(store, story.WithObjectStorage(storage))
	h := wrapHandler(svc)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "cover.png")
	_, _ = fw.Write([]byte("fake-image"))
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/stories/s1/cover", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["id"] == "" {
		t.Fatal("expected asset id")
	}
	if resp["type"] != story.AssetTypeCover {
		t.Fatalf("expected COVER, got %v", resp["type"])
	}
}

func TestSearchStoriesHandlerReturnsPublic(t *testing.T) {
	store := newFakeStore()
	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A", Visibility: story.VisibilityPublic}
	store.stories["s2"] = story.Story{ID: "s2", Slug: "b", Title: "B", Visibility: story.VisibilityPrivate}
	store.storyGenres["s1"] = []string{"fantasy"}
	svc := story.NewService(store)
	h := wrapHandler(svc)

	rec := doJSON(t, h, http.MethodGet, "/api/v1/stories?q=road&genre=fantasy&sort=NEW", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	stories, ok := resp["stories"].([]any)
	if !ok {
		t.Fatalf("expected stories array, got %v", resp["stories"])
	}
	if len(stories) != 1 {
		t.Fatalf("expected 1 public story, got %d", len(stories))
	}
}
