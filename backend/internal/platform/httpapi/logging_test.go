package httpapi_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/synaudio/synaudio/backend/internal/platform/httpapi"
)

func TestRequestLoggerEmitsStructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(httpapi.WithRequestLogger(logger))
	r.Post("/api/v1/stories", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/stories", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	entry := decodeLog(t, buf.Bytes())
	if entry["method"] != "POST" {
		t.Fatalf("expected method POST, got %v", entry["method"])
	}
	if entry["route"] != "/api/v1/stories" {
		t.Fatalf("expected route, got %v", entry["route"])
	}
	if entry["status"] != float64(201) {
		t.Fatalf("expected status 201, got %v", entry["status"])
	}
	if _, ok := entry["latency_ms"]; !ok {
		t.Fatal("expected latency_ms field in log entry")
	}
	if entry["correlation_id"] == "" {
		t.Fatal("expected correlation_id in log entry")
	}
	if entry["request_id"] == "" {
		t.Fatal("expected request_id in log entry")
	}
}

func TestRequestLoggerNeverLogsAuthorizationHeader(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := httpapi.WithRequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer super-secret-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if bytes.Contains(buf.Bytes(), []byte("super-secret-token")) {
		t.Fatal("log output must not contain the authorization token")
	}
	if bytes.Contains(buf.Bytes(), []byte("Authorization")) {
		t.Fatal("log output must not contain the Authorization header name")
	}
}

func TestRequestLoggerUsesValidatedCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := httpapi.WithRequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("X-Correlation-ID", "client-trace-01")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	entry := decodeLog(t, buf.Bytes())
	if entry["correlation_id"] != "client-trace-01" {
		t.Fatalf("expected validated correlation id, got %v", entry["correlation_id"])
	}
}

func TestRequestLoggerRejectsUnsafeCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(httpapi.WithRequestLogger(logger))
	r.Get("/api/v1/me", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("X-Correlation-ID", "unsafe value with spaces")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	entry := decodeLog(t, buf.Bytes())
	if entry["correlation_id"] != entry["request_id"] {
		t.Fatalf("expected unsafe correlation to fall back to request_id, got correlation=%v request_id=%v", entry["correlation_id"], entry["request_id"])
	}
}

func TestRequestLoggerRecoveredPanicEmitsSafeErrorRecord(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(httpapi.WithRequestLogger(logger))
	r.Get("/api/v1/panic", func(_ http.ResponseWriter, _ *http.Request) {
		panic("super-secret-token must never appear")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/panic", nil)
	req.Header.Set("Authorization", "Bearer super-secret-token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if !strings.Contains(buf.String(), "request panic recovered") {
		t.Fatalf("expected panic log record, got %s", buf.String())
	}
	if strings.Contains(buf.String(), "super-secret-token") {
		t.Fatal("panic log must not contain panic value or auth token")
	}

	entry := decodeLog(t, buf.Bytes())
	if entry["status"] != float64(http.StatusInternalServerError) {
		t.Fatalf("expected status 500, got %v", entry["status"])
	}
	if entry["correlation_id"] == "" || entry["request_id"] == "" {
		t.Fatalf("expected correlated panic record, got %v", entry)
	}
}

func decodeLog(t *testing.T, raw []byte) map[string]any {
	lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
	if len(lines) == 0 {
		t.Fatal("no log output")
	}
	var entry map[string]any
	if err := json.Unmarshal(lines[len(lines)-1], &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v\n%s", err, raw)
	}
	return entry
}
