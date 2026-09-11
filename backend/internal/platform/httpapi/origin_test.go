package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
	"github.com/synaudio/synaudio/backend/internal/platform/httpapi"
)

func originProbe(cfg httpapi.TrustedProxyConfig) http.Handler {
	return httpapi.WithTrustedClientIP(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Seen-Forwarded-Proto", r.Header.Get("X-Forwarded-Proto"))
		w.WriteHeader(http.StatusNoContent)
	}))
}

func refreshRequest(origin string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "http://app.example.com/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: identity.DevelopmentRefreshCookieName, Value: "refresh-token"})
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	return req
}

func TestCookieRefreshRequiresSameOrigin(t *testing.T) {
	h := originProbe(httpapi.TrustedProxyConfig{})

	missing := httptest.NewRecorder()
	h.ServeHTTP(missing, refreshRequest(""))
	if missing.Code != http.StatusForbidden || !containsBody(missing.Body.String(), "CSRF_ORIGIN_REQUIRED") {
		t.Fatalf("missing Origin: status=%d body=%s", missing.Code, missing.Body.String())
	}

	crossOrigin := httptest.NewRecorder()
	h.ServeHTTP(crossOrigin, refreshRequest("https://evil.example"))
	if crossOrigin.Code != http.StatusForbidden || !containsBody(crossOrigin.Body.String(), "CSRF_ORIGIN_MISMATCH") {
		t.Fatalf("cross Origin: status=%d body=%s", crossOrigin.Code, crossOrigin.Body.String())
	}

	sameOrigin := httptest.NewRecorder()
	h.ServeHTTP(sameOrigin, refreshRequest("http://app.example.com"))
	if sameOrigin.Code != http.StatusNoContent {
		t.Fatalf("same Origin: status=%d body=%s", sameOrigin.Code, sameOrigin.Body.String())
	}
}

func TestUntrustedForwardedProtoCannotUpgradeCookieOrigin(t *testing.T) {
	h := originProbe(httpapi.TrustedProxyConfig{})
	req := refreshRequest("https://app.example.com")
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("X-Forwarded-Proto", "https")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || !containsBody(rec.Body.String(), "CSRF_ORIGIN_MISMATCH") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Seen-Forwarded-Proto"); got != "" {
		t.Fatalf("untrusted forwarded proto leaked downstream: %q", got)
	}
}

func TestTrustedProxyMaySupplyForwardedHTTPS(t *testing.T) {
	cfg, err := httpapi.ParseTrustedProxyConfig("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	h := originProbe(cfg)
	req := refreshRequest("https://app.example.com")
	req.RemoteAddr = "10.1.2.3:4321"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-For", "198.51.100.25")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Seen-Forwarded-Proto"); got != "https" {
		t.Fatalf("trusted forwarded proto = %q, want https", got)
	}
}

func TestBearerLogoutIsNotBlockedByCookieOriginGuard(t *testing.T) {
	h := originProbe(httpapi.TrustedProxyConfig{})
	req := httptest.NewRequest(http.MethodPost, "http://app.example.com/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: identity.DevelopmentRefreshCookieName, Value: "refresh-token"})
	req.Header.Set("Authorization", "Bearer access-token")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func containsBody(body, want string) bool {
	for i := 0; i+len(want) <= len(body); i++ {
		if body[i:i+len(want)] == want {
			return true
		}
	}
	return false
}
