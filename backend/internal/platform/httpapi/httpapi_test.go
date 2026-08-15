package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/platform/httpapi"
)

func TestHealthReturnsOKWithoutDependencies(t *testing.T) {
	handler := httpapi.NewRouter(httpapi.Dependencies{
		ReadyCheck: func() error {
			t.Fatal("health must not call ready checks")
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %#v", body)
	}
}

func TestReadyReturnsServiceUnavailableWhenDBUnavailable(t *testing.T) {
	handler := httpapi.NewRouter(httpapi.Dependencies{
		ReadyCheck: func() error {
			return httpapi.ErrDependencyUnavailable
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestReadyReturnsOKWhenDependenciesReady(t *testing.T) {
	handler := httpapi.NewRouter(httpapi.Dependencies{
		ReadyCheck: func() error { return nil },
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
