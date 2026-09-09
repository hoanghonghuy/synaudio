package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/platform/httpapi"
)

func TestClientIPFromRemoteAddrWithoutProxyTrust(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.10:4321"
	req.Header.Set("X-Real-IP", "198.51.100.1")
	req.Header.Set("X-Forwarded-For", "198.51.100.1, 203.0.113.1")

	if got := httpapi.ClientIP(req); got != "203.0.113.10" {
		t.Fatalf("expected 203.0.113.10, got %q", got)
	}
}

func TestClientIPUsesXRealIPFromTrustedProxy(t *testing.T) {
	cfg, err := httpapi.ParseTrustedProxyConfig("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.1.2.3:8080"
	req.Header.Set("X-Real-IP", "198.51.100.42")

	handler := httpapi.WithTrustedClientIP(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(httpapi.ClientIP(r)))
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "198.51.100.42" {
		t.Fatalf("expected X-Real-IP client, got %q", rec.Body.String())
	}
}
