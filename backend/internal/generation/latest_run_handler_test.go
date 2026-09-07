package generation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestLatestChapterGenerationRunEndpointReturnsDurableRun(t *testing.T) {
	store := newFakeStore()
	store.runs["story-1"] = []GenerationRun{
		{ID: "run-1", StoryID: "story-1", ChapterID: "chapter-1", RunType: "CHAPTER_GENERATION", Status: "RUNNING"},
	}
	handler := NewLatestRunAwareHandler(chi.NewRouter(), NewService(store))

	req := httptest.NewRequest(http.MethodGet, "/admin/chapters/chapter-1/generation-run/latest", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got GenerationRun
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "run-1" || got.Status != "RUNNING" {
		t.Fatalf("unexpected run: %#v", got)
	}
}

func TestLatestChapterGenerationRunEndpointReturnsNotFoundWithoutSideEffect(t *testing.T) {
	store := newFakeStore()
	handler := NewLatestRunAwareHandler(chi.NewRouter(), NewService(store))

	req := httptest.NewRequest(http.MethodGet, "/admin/chapters/chapter-1/generation-run/latest", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.runs) != 0 {
		t.Fatalf("latest-run read must not create runs, got %#v", store.runs)
	}
}

func TestChapterContentProjectionIncludesRunBeforeRevisionExists(t *testing.T) {
	store := newFakeStore()
	store.runs["story-1"] = []GenerationRun{
		{ID: "run-1", StoryID: "story-1", ChapterID: "chapter-1", RunType: "CHAPTER_GENERATION", Status: "PENDING"},
	}
	handler := NewLatestRunAwareHandler(chi.NewRouter(), NewService(store))

	req := httptest.NewRequest(http.MethodGet, "/admin/chapters/chapter-1/content", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Revisions     []ContentRevision `json:"revisions"`
		GenerationRun *GenerationRun    `json:"generation_run"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Revisions) != 0 {
		t.Fatalf("expected no revisions yet, got %#v", got.Revisions)
	}
	if got.GenerationRun == nil || got.GenerationRun.ID != "run-1" {
		t.Fatalf("expected durable run projection, got %#v", got.GenerationRun)
	}
}

func TestChapterContentProjectionExplicitlyReturnsNoRun(t *testing.T) {
	store := newFakeStore()
	handler := NewLatestRunAwareHandler(chi.NewRouter(), NewService(store))

	req := httptest.NewRequest(http.MethodGet, "/admin/chapters/chapter-1/content", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got["generation_run"] != nil {
		t.Fatalf("expected explicit null generation_run, got %#v", got["generation_run"])
	}
}

func TestLatestRunAwareRoutesKeepProjectionWhenFlattened(t *testing.T) {
	store := newFakeStore()
	store.runs["story-1"] = []GenerationRun{
		{ID: "run-1", StoryID: "story-1", ChapterID: "chapter-1", RunType: "CHAPTER_GENERATION", Status: "RUNNING"},
	}
	service := NewService(store)
	source := NewLatestRunAwareHandler(NewHandler(service, nil), service)
	routes, ok := source.(chi.Routes)
	if !ok {
		t.Fatal("latest-run-aware handler must expose chi routes for API composition")
	}

	flattened := chi.NewRouter()
	if err := chi.Walk(routes, func(method, route string, handler http.Handler, _ ...func(http.Handler) http.Handler) error {
		flattened.Method(method, route, handler)
		return nil
	}); err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/chapters/chapter-1/content", nil)
	rec := httptest.NewRecorder()
	flattened.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 through flattened routes, got %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		GenerationRun *GenerationRun `json:"generation_run"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.GenerationRun == nil || got.GenerationRun.ID != "run-1" {
		t.Fatalf("projection was lost during route flattening: %#v", got.GenerationRun)
	}
}
