package audio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminPreviewRemainsIndependentFromListenerEligibility(t *testing.T) {
	store := newFakeStore()
	svc := newTestService(
		store,
		WithPresigner(fakePresigner{url: "https://cdn.example.com"}),
		WithListenerAudioGate(&fakeListenerAudioGate{err: ErrListenerAudioNotEligible}),
	)
	handler := NewHandler(svc)

	narration, err := svc.CreateNarrationRevision(t.Context(), "chapter-1", "cr-1", "voice-1", "Hello.", "u1")
	if err != nil {
		t.Fatalf("create narration: %v", err)
	}
	asset, err := svc.CreateAudioAsset(t.Context(), "chapter-1", narration.ID, "audio/chapter-1/v1.mp3", "audio/mpeg", 100, 1000, 128)
	if err != nil {
		t.Fatalf("create audio asset: %v", err)
	}

	adminReq := httptest.NewRequest(http.MethodGet, "/admin/chapters/chapter-1/audio/"+asset.ID+"/preview-url", nil)
	adminRec := httptest.NewRecorder()
	handler.ServeHTTP(adminRec, adminReq)
	if adminRec.Code != http.StatusOK {
		t.Fatalf("expected admin preview 200 for unpublished READY asset, got %d: %s", adminRec.Code, adminRec.Body.String())
	}
	var adminBody map[string]string
	if err := json.NewDecoder(adminRec.Body).Decode(&adminBody); err != nil {
		t.Fatalf("decode admin preview: %v", err)
	}
	if adminBody["url"] != "https://cdn.example.com/audio/chapter-1/v1.mp3" {
		t.Fatalf("unexpected admin preview url: %q", adminBody["url"])
	}

	// Make the exact same durable asset active so the public path reaches its
	// independent listener-eligibility gate rather than failing only because no
	// active asset exists. The gate still denies this unpublished chapter.
	if _, err := svc.ActivateAudioAsset(t.Context(), "chapter-1", asset.ID); err != nil {
		t.Fatalf("activate audio asset: %v", err)
	}

	publicReq := httptest.NewRequest(http.MethodGet, "/chapters/chapter-1/audio-url", nil)
	publicRec := httptest.NewRecorder()
	handler.ServeHTTP(publicRec, publicReq)
	if publicRec.Code != http.StatusNotFound {
		t.Fatalf("expected public listener URL to remain denied for ineligible chapter, got %d: %s", publicRec.Code, publicRec.Body.String())
	}
}
