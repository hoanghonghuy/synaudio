package httpapi_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/platform/httpapi"
)

func TestRequestLoggerEmitsStructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := httpapi.WithRequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/stories", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v\n%s", err, buf.String())
	}

	if entry["method"] != "POST" {
		t.Fatalf("expected method POST, got %v", entry["method"])
	}
	if entry["path"] != "/api/v1/stories" {
		t.Fatalf("expected path, got %v", entry["path"])
	}
	if entry["status"] != float64(201) {
		t.Fatalf("expected status 201, got %v", entry["status"])
	}
	if _, ok := entry["latency_ms"]; !ok {
		t.Fatal("expected latency_ms field in log entry")
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
