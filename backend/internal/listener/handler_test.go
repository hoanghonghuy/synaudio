package listener

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListenerHandlerUsesResolvedUserForFavorites(t *testing.T) {
	store := newFakeStore()
	handler := NewHandler(NewService(store), func(context.Context, *http.Request) (string, error) {
		return "session-user", nil
	})

	req := httptest.NewRequest(http.MethodPut, "/me/favorites/story-1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !store.favorites["session-user"]["story-1"] {
		t.Fatal("expected favorite to belong to resolved session user")
	}
}

func TestListenerHandlerRejectsMissingSession(t *testing.T) {
	store := newFakeStore()
	handler := NewHandler(NewService(store), func(context.Context, *http.Request) (string, error) {
		return "", errors.New("missing session")
	})

	req := httptest.NewRequest(http.MethodGet, "/me/favorites", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListenerHandlerIgnoresUserIDHeader(t *testing.T) {
	store := newFakeStore()
	handler := NewHandler(NewService(store), func(context.Context, *http.Request) (string, error) {
		return "session-user", nil
	})

	var body bytes.Buffer
	_ = json.NewEncoder(&body).Encode(map[string]any{
		"position_ms":         1000,
		"audio_asset_id":      "asset-1",
		"playback_session_id": "playback-1",
	})
	req := httptest.NewRequest(http.MethodPut, "/me/progress/chapter-1", &body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "other-user")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, ok := store.progress["other-user"]; ok {
		t.Fatal("must not write progress for X-User-ID header")
	}
	if _, ok := store.progress["session-user"]; !ok {
		t.Fatal("expected progress for resolved session user")
	}
}
