package httpapi_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/platform/httpapi"
)

func TestReadyReportsPerDependencyStatus(t *testing.T) {
	handler := httpapi.NewRouter(httpapi.Dependencies{
		ReadyCheck: func() error { return nil },
		DependencyChecks: map[string]func() error{
			"database": func() error { return nil },
			"storage":  func() error { return errors.New("storage down") },
			"text_ai":  func() error { return nil },
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	deps, ok := body["dependencies"].(map[string]any)
	if !ok {
		t.Fatalf("expected dependencies map, got %#v", body)
	}
	if deps["database"] != "ok" {
		t.Fatalf("expected database ok, got %v", deps["database"])
	}
	if deps["storage"] != "unavailable" {
		t.Fatalf("expected storage unavailable, got %v", deps["storage"])
	}
}

func TestReadyReturnsOKWhenAllDependenciesReady(t *testing.T) {
	handler := httpapi.NewRouter(httpapi.Dependencies{
		ReadyCheck: func() error { return nil },
		DependencyChecks: map[string]func() error{
			"database": func() error { return nil },
			"storage":  func() error { return nil },
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestReadyWithoutDependencyChecksStillWorks(t *testing.T) {
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
