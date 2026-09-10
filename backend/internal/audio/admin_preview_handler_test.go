package audio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminAudioPreviewRouteReturnsPresignedURL(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(store, WithPresigner(fakePresigner{url: "https://cdn.example.com"}))
	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")
	asset, _ := svc.CreateAudioAsset(context.Background(), "c1", nar.ID, "audio/c1/v1.mp3", "audio/mpeg", 100, 1000, 128)

	req := httptest.NewRequest(http.MethodGet, "/admin/chapters/c1/audio/"+asset.ID+"/preview-url", nil)
	res := httptest.NewRecorder()
	NewHandler(svc).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["url"] != "https://cdn.example.com/audio/c1/v1.mp3" {
		t.Fatalf("unexpected url: %q", body["url"])
	}
}

func TestAdminAudioPreviewRouteConcealsCrossChapterAsset(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(store, WithPresigner(fakePresigner{url: "https://cdn.example.com"}))
	nar, _ := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "Hello.", "u1")
	asset, _ := svc.CreateAudioAsset(context.Background(), "c1", nar.ID, "audio/c1/v1.mp3", "audio/mpeg", 100, 1000, 128)

	req := httptest.NewRequest(http.MethodGet, "/admin/chapters/c2/audio/"+asset.ID+"/preview-url", nil)
	res := httptest.NewRecorder()
	NewHandler(svc).ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", res.Code, res.Body.String())
	}
}
