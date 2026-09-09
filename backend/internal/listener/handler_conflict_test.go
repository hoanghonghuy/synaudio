package listener

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSaveProgressHandlerReturnsConflictWithAuthoritativeState(t *testing.T) {
	store := newCASFakeStore()
	svc := NewService(store)
	handler := NewHandler(svc, func(context.Context, *http.Request) (string, error) {
		return "session-user", nil
	})

	seedBody := map[string]any{
		"position_ms":         5000,
		"audio_asset_id":      "asset-1",
		"playback_session_id": "session-1",
		"expected_version":    0,
	}
	var seed bytes.Buffer
	_ = json.NewEncoder(&seed).Encode(seedBody)
	seedReq := httptest.NewRequest(http.MethodPut, "/me/progress/chapter-1", &seed)
	seedReq.Header.Set("Content-Type", "application/json")
	seedRec := httptest.NewRecorder()
	handler.ServeHTTP(seedRec, seedReq)
	if seedRec.Code != http.StatusOK {
		t.Fatalf("seed save expected 200, got %d: %s", seedRec.Code, seedRec.Body.String())
	}

	var seeded ListeningProgress
	if err := json.NewDecoder(seedRec.Body).Decode(&seeded); err != nil {
		t.Fatalf("decode seed response: %v", err)
	}

	authoritativeBody := map[string]any{
		"position_ms":         9000,
		"audio_asset_id":      "asset-1",
		"playback_session_id": "session-2",
		"expected_version":    seeded.Version,
	}
	var authoritative bytes.Buffer
	_ = json.NewEncoder(&authoritative).Encode(authoritativeBody)
	authReq := httptest.NewRequest(http.MethodPut, "/me/progress/chapter-1", &authoritative)
	authReq.Header.Set("Content-Type", "application/json")
	authRec := httptest.NewRecorder()
	handler.ServeHTTP(authRec, authReq)
	if authRec.Code != http.StatusOK {
		t.Fatalf("authoritative save expected 200, got %d: %s", authRec.Code, authRec.Body.String())
	}

	staleBody := map[string]any{
		"position_ms":         1000,
		"audio_asset_id":      "asset-1",
		"playback_session_id": "session-stale",
		"expected_version":    seeded.Version,
	}
	var stale bytes.Buffer
	_ = json.NewEncoder(&stale).Encode(staleBody)
	staleReq := httptest.NewRequest(http.MethodPut, "/me/progress/chapter-1", &stale)
	staleReq.Header.Set("Content-Type", "application/json")
	staleRec := httptest.NewRecorder()
	handler.ServeHTTP(staleRec, staleReq)

	if staleRec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", staleRec.Code, staleRec.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(staleRec.Body).Decode(&body); err != nil {
		t.Fatalf("decode conflict response: %v", err)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "PROGRESS_VERSION_CONFLICT" {
		t.Fatalf("expected PROGRESS_VERSION_CONFLICT, got %#v", errObj["code"])
	}
	progress, ok := body["progress"].(map[string]any)
	if !ok {
		t.Fatalf("expected progress payload, got %#v", body["progress"])
	}
	if int64(progress["PositionMs"].(float64)) != 9000 {
		t.Fatalf("expected authoritative position 9000, got %#v", progress["PositionMs"])
	}
}
