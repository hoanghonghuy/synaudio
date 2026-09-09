package audio

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetLatestNarrationRevisionEndpointReturnsNewestRevision(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	handler := NewHandler(svc)

	first := NarrationRevision{ID: "nar-1", ChapterID: "chapter-1", RevisionNo: 1, Script: "one", Status: "DRAFT"}
	second := NarrationRevision{ID: "nar-2", ChapterID: "chapter-1", RevisionNo: 2, Script: "two", Status: "DRAFT"}
	store.narrations["chapter-1"] = []NarrationRevision{first, second}

	req := httptest.NewRequest(http.MethodGet, "/admin/chapters/chapter-1/narration/latest", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got NarrationRevision
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "nar-2" || got.RevisionNo != 2 {
		t.Fatalf("unexpected narration: %#v", got)
	}
}

func TestGetLatestNarrationRevisionEndpointReturnsNotFoundWithoutSideEffect(t *testing.T) {
	handler := NewHandler(NewService(newFakeStore()))

	req := httptest.NewRequest(http.MethodGet, "/admin/chapters/chapter-1/narration/latest", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetLatestReadyAudioAssetForNarrationEndpointReturnsNewestReadyCandidate(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	handler := NewHandler(svc)

	store.narrations["chapter-1"] = []NarrationRevision{{ID: "nar-1", ChapterID: "chapter-1", RevisionNo: 1, Status: "DRAFT"}}
	store.assets["chapter-1"] = []AudioAsset{
		{ID: "asset-1", ChapterID: "chapter-1", VersionNo: 1, SourceNarrationRevisionID: "nar-1", Status: "READY", IsActive: true},
		{ID: "asset-2", ChapterID: "chapter-1", VersionNo: 2, SourceNarrationRevisionID: "nar-1", Status: "READY", IsActive: false, Checksum: "abc"},
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/chapters/chapter-1/narration/nar-1/audio/latest-ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var got AudioAsset
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "asset-2" || got.Checksum != "abc" || got.IsActive {
		t.Fatalf("unexpected ready asset: %#v", got)
	}
}

func TestGetLatestReadyAudioAssetForNarrationEndpointReturnsNotFoundForStaleOlderNarrationAsset(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	handler := NewHandler(svc)

	store.narrations["chapter-1"] = []NarrationRevision{
		{ID: "nar-1", ChapterID: "chapter-1", RevisionNo: 1, Status: "DRAFT"},
		{ID: "nar-2", ChapterID: "chapter-1", RevisionNo: 2, Status: "DRAFT"},
	}
	store.assets["chapter-1"] = []AudioAsset{
		{ID: "asset-1", ChapterID: "chapter-1", VersionNo: 1, SourceNarrationRevisionID: "nar-1", Status: "READY", IsActive: false},
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/chapters/chapter-1/narration/nar-2/audio/latest-ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for latest narration without matching READY asset, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestActivateAudioAssetEndpointRejectsStaleReadyFromOlderNarration(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	handler := NewHandler(svc)

	store.narrations["chapter-1"] = []NarrationRevision{
		{ID: "nar-1", ChapterID: "chapter-1", RevisionNo: 1, Status: "DRAFT"},
		{ID: "nar-2", ChapterID: "chapter-1", RevisionNo: 2, Status: "DRAFT"},
	}
	store.assets["chapter-1"] = []AudioAsset{
		{ID: "asset-1", ChapterID: "chapter-1", VersionNo: 1, SourceNarrationRevisionID: "nar-1", Status: "READY", IsActive: false},
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/chapters/chapter-1/audio/asset-1/activate", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for stale activation, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.assets["chapter-1"][0].IsActive {
		t.Fatal("stale activation must not promote asset")
	}
}

func TestSynthesizeNarrationEndpointRejectsChapterMismatch(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(
		store,
		WithTTS(NewMockTTS()),
		WithObjectStorage(newFakeObjectStorage()),
		WithAudioProcessor(NewMockAudioProcessor()),
	)
	handler := NewHandler(svc)

	nar, err := svc.CreateNarrationRevision(t.Context(), "chapter-a", "cr-1", "voice-1", "Hello world.", "u1")
	if err != nil {
		t.Fatalf("create narration: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/chapters/chapter-b/narration/"+nar.ID+"/synthesize", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(store.assets["chapter-a"]) != 0 || len(store.assets["chapter-b"]) != 0 {
		t.Fatal("chapter mismatch must not synthesize audio")
	}
}

func TestSynthesizeNarrationEndpointHappyPathReturnsReadyAsset(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(
		store,
		WithTTS(NewMockTTS()),
		WithObjectStorage(newFakeObjectStorage()),
		WithAudioProcessor(NewMockAudioProcessor()),
	)
	handler := NewHandler(svc)

	nar, err := svc.CreateNarrationRevision(t.Context(), "chapter-1", "cr-1", "voice-1", "First sentence. Second sentence.", "u1")
	if err != nil {
		t.Fatalf("create narration: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/chapters/chapter-1/narration/"+nar.ID+"/synthesize", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var got AudioAsset
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Status != "READY" || got.IsActive {
		t.Fatalf("expected durable READY non-active asset, got %#v", got)
	}
	if got.SourceNarrationRevisionID != nar.ID || got.ChapterID != "chapter-1" {
		t.Fatalf("unexpected asset binding: %#v", got)
	}
	if got.Checksum == "" || got.StorageKey == "" {
		t.Fatalf("expected checksum and storage metadata, got %#v", got)
	}
}

func TestSynthesizeNarrationForChapterRejectsMissingNarration(t *testing.T) {
	svc := NewService(newFakeStore())
	if _, err := svc.SynthesizeNarrationForChapter(t.Context(), "chapter-1", "missing"); !errors.Is(err, ErrNarrationNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
