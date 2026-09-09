package httpapi_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/platform/httpapi"
)

func TestLimitRequestBodyRejectsOversizedContentLength(t *testing.T) {
	handler := httpapi.LimitRequestBody(32)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	body := strings.Repeat("a", 64)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", strings.NewReader(body))
	req.ContentLength = int64(len(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
	assertErrorCode(t, rec.Body.Bytes(), "REQUEST_TOO_LARGE")
}

func TestLimitRequestBodyRejectsOversizedChunkedBody(t *testing.T) {
	handler := httpapi.LimitRequestBody(32)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	body := strings.Repeat("a", 64)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", strings.NewReader(body))
	req.ContentLength = -1
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
	assertErrorCode(t, rec.Body.Bytes(), "REQUEST_TOO_LARGE")
}

func TestLimitRequestBodyAllowsSmallJSONPayload(t *testing.T) {
	handler := httpapi.LimitRequestBody(128)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", strings.NewReader(`{"name":"ok"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestLimitRequestBodySkipsMethodsWithoutBodies(t *testing.T) {
	handler := httpapi.LimitRequestBody(1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRouterAppliesRequestBodyLimit(t *testing.T) {
	handler := httpapi.NewRouter(httpapi.Dependencies{})

	body := strings.Repeat("x", httpapi.DefaultMaxRequestBodyBytes+1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.ContentLength = int64(len(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d body=%s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), "REQUEST_TOO_LARGE")
}

func assertErrorCode(t *testing.T, body []byte, code string) {
	t.Helper()
	var payload map[string]map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode error body: %v raw=%s", err, string(body))
	}
	if payload["error"]["code"] != code {
		t.Fatalf("expected error code %q, got %#v", code, payload["error"])
	}
}

func TestLimitRequestBodyDrainsRequestBodyOnSuccess(t *testing.T) {
	handler := httpapi.LimitRequestBody(128)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", strings.NewReader(`{"ok":true}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}
