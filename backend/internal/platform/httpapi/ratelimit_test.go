package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/platform/httpapi"
)

func TestRateLimitAllowsUnderLimit(t *testing.T) {
	limiter := httpapi.NewRateLimiter(3, 60)

	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stories", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rec.Code)
		}
	}
}

func TestRateLimitBlocksOverLimit(t *testing.T) {
	limiter := httpapi.NewRateLimiter(2, 60)

	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stories", nil)
		req.RemoteAddr = "192.0.2.2:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stories", nil)
	req.RemoteAddr = "192.0.2.2:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}

func TestRateLimitIsPerClient(t *testing.T) {
	limiter := httpapi.NewRateLimiter(1, 60)

	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Client A uses its single request.
	reqA := httptest.NewRequest(http.MethodGet, "/api/v1/stories", nil)
	reqA.RemoteAddr = "192.0.2.3:1234"
	recA := httptest.NewRecorder()
	handler.ServeHTTP(recA, reqA)

	// Client B is unaffected.
	reqB := httptest.NewRequest(http.MethodGet, "/api/v1/stories", nil)
	reqB.RemoteAddr = "192.0.2.4:1234"
	recB := httptest.NewRecorder()
	handler.ServeHTTP(recB, reqB)

	if recB.Code != http.StatusOK {
		t.Fatalf("client B expected 200, got %d", recB.Code)
	}
}
