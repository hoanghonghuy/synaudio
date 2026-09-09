package workerprobe_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/synaudio/synaudio/backend/internal/platform/workerprobe"
)

func TestHealthReturnsOKWithoutDependencies(t *testing.T) {
	handler := workerprobe.NewHandler(workerprobe.Dependencies{
		PingDatabase: func(context.Context) error {
			t.Fatal("health must not call database")
			return nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	assertStatus(t, rec, "ok")
}

func TestReadyReturnsOKWhenDatabaseAndHeartbeatFresh(t *testing.T) {
	handler := workerprobe.NewHandler(workerprobe.Dependencies{
		PingDatabase:  func(context.Context) error { return nil },
		HeartbeatAge:  func() time.Duration { return 2 * time.Second },
		AcceptingWork: func() bool { return true },
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertStatus(t, rec, "ready")
}

func TestReadyReturnsServiceUnavailableWhenDatabaseUnavailable(t *testing.T) {
	handler := workerprobe.NewHandler(workerprobe.Dependencies{
		PingDatabase: func(context.Context) error { return errors.New("db down") },
		HeartbeatAge: func() time.Duration { return time.Second },
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	assertStatus(t, rec, "degraded")
}

func TestReadyReturnsServiceUnavailableWhenHeartbeatStale(t *testing.T) {
	handler := workerprobe.NewHandler(workerprobe.Dependencies{
		PingDatabase:    func(context.Context) error { return nil },
		HeartbeatAge:    func() time.Duration { return 2 * time.Minute },
		MaxHeartbeatAge: 60 * time.Second,
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	assertStatus(t, rec, "degraded")
}

func TestReadyReturnsDrainingWhenNotAcceptingWork(t *testing.T) {
	handler := workerprobe.NewHandler(workerprobe.Dependencies{
		PingDatabase:  func(context.Context) error { return nil },
		HeartbeatAge:  func() time.Duration { return time.Second },
		AcceptingWork: func() bool { return false },
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
	assertStatus(t, rec, "draining")
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != want {
		t.Fatalf("expected status %q, got %#v", want, body)
	}
}
