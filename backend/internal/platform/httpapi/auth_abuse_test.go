package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/synaudio/synaudio/backend/internal/platform/httpapi"
	"github.com/synaudio/synaudio/backend/internal/platform/metrics"
)

func TestAuthAbuseAllowsTrafficUnderLimit(t *testing.T) {
	limiter := httpapi.NewMemoryAbuseLimiter()
	mw := httpapi.NewAuthAbuseMiddleware(httpapi.DefaultAuthAbusePolicies(), limiter, nil, nil)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"a@example.com","password":"secret"}`))
		req.RemoteAddr = "192.0.2.10:1234"
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d: %s", i, rec.Code, rec.Body.String())
		}
	}
}

func TestAuthAbuseBlocksLoginOverClientLimit(t *testing.T) {
	policies := httpapi.DefaultAuthAbusePolicies()
	policies["POST /login"].Limits[0].Limit = 2

	limiter := httpapi.NewMemoryAbuseLimiter()
	mw := httpapi.NewAuthAbuseMiddleware(policies, limiter, nil, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"a@example.com","password":"secret"}`))
		req.RemoteAddr = "192.0.2.11:1234"
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"a@example.com","password":"secret"}`))
	req.RemoteAddr = "192.0.2.11:1234"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header on 429")
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "RATE_LIMITED" {
		t.Fatalf("expected RATE_LIMITED, got %v", errObj["code"])
	}
}

func TestAuthAbusePasswordForgotStaysEnumerationSafeWhenThrottled(t *testing.T) {
	policies := httpapi.DefaultAuthAbusePolicies()
	policies["POST /password/forgot"].Limits[0].Limit = 1

	limiter := httpapi.NewMemoryAbuseLimiter()
	mw := httpapi.NewAuthAbuseMiddleware(policies, limiter, nil, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"accepted"}`))
	}))

	body := `{"email":"victim@example.com"}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/password/forgot", strings.NewReader(body))
		req.RemoteAddr = "192.0.2.12:1234"
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("request %d: expected 202, got %d: %s", i, rec.Code, rec.Body.String())
		}
	}
}

func TestAuthAbuseEmailResendStaysEnumerationSafeWhenThrottled(t *testing.T) {
	policies := httpapi.DefaultAuthAbusePolicies()
	policies["POST /email/resend"].Limits[0].Limit = 1

	limiter := httpapi.NewMemoryAbuseLimiter()
	mw := httpapi.NewAuthAbuseMiddleware(policies, limiter, nil, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"accepted"}`))
	}))

	body := `{"email":"reader@example.com"}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/email/resend", strings.NewReader(body))
		req.RemoteAddr = "192.0.2.13:1234"
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("request %d: expected 202, got %d: %s", i, rec.Code, rec.Body.String())
		}
	}
}

func TestAuthAbuseAccountDimensionBlocksDistributedLoginAttempts(t *testing.T) {
	policies := httpapi.DefaultAuthAbusePolicies()
	for i := range policies["POST /login"].Limits {
		if policies["POST /login"].Limits[i].Dimension == httpapi.AbuseDimensionAccount {
			policies["POST /login"].Limits[i].Limit = 2
		}
	}

	limiter := httpapi.NewMemoryAbuseLimiter()
	mw := httpapi.NewAuthAbuseMiddleware(policies, limiter, nil, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ips := []string{"192.0.2.21:1234", "192.0.2.22:1234"}
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"target@example.com","password":"secret"}`))
		req.RemoteAddr = ips[i]
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"target@example.com","password":"secret"}`))
	req.RemoteAddr = "192.0.2.99:1234"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 from account dimension, got %d", rec.Code)
	}
}

func TestAuthAbuseWindowResetsAfterExpiry(t *testing.T) {
	policies := httpapi.DefaultAuthAbusePolicies()
	policies["POST /register"].Limits[0].Limit = 1
	policies["POST /register"].Limits[0].Window = 50 * time.Millisecond

	limiter := httpapi.NewMemoryAbuseLimiter()
	mw := httpapi.NewAuthAbuseMiddleware(policies, limiter, nil, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"a@example.com","password":"secret"}`))
	req.RemoteAddr = "192.0.2.20:1234"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first request: expected 201, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"b@example.com","password":"secret"}`))
	req.RemoteAddr = "192.0.2.20:1234"
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: expected 429, got %d", rec.Code)
	}

	time.Sleep(60 * time.Millisecond)

	req = httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"c@example.com","password":"secret"}`))
	req.RemoteAddr = "192.0.2.20:1234"
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("after window reset: expected 201, got %d", rec.Code)
	}
}

func TestTrustedProxyIgnoresForwardedHeadersWhenNotConfigured(t *testing.T) {
	cfg := httpapi.TrustedProxyConfig{}
	handler := httpapi.WithTrustedClientIP(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(httpapi.ClientIP(r)))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.9")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "203.0.113.5" {
		t.Fatalf("expected direct remote addr, got %q", rec.Body.String())
	}
}

func TestTrustedProxyUsesForwardedHeadersFromTrustedPeer(t *testing.T) {
	cfg, err := httpapi.ParseTrustedProxyConfig("127.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	handler := httpapi.WithTrustedClientIP(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(httpapi.ClientIP(r)))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.9")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "198.51.100.9" {
		t.Fatalf("expected forwarded client IP, got %q", rec.Body.String())
	}
}

func TestRouterMountsAuthAbuseOnSensitiveAuthRoutes(t *testing.T) {
	policies := httpapi.DefaultAuthAbusePolicies()
	policies["POST /login"].Limits[0].Limit = 1

	limiter := httpapi.NewMemoryAbuseLimiter()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	registry := metrics.NewRegistry()

	auth := chi.NewRouter()
	auth.Post("/login", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := httpapi.NewRouter(httpapi.Dependencies{
		Logger:      log,
		AuthHandler: auth,
		AuthAbuse: &httpapi.AuthAbuse{
			Policies: policies,
			Limiter:  limiter,
			Metrics:  registry,
			Logger:   log,
		},
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"a@example.com","password":"x"}`))
		req.RemoteAddr = "192.0.2.30:1234"
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if i == 0 && rec.Code != http.StatusOK {
			t.Fatalf("first login: expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		if i == 1 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("second login: expected 429, got %d: %s", rec.Code, rec.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health should remain unthrottled, got %d", rec.Code)
	}
}

func TestAuthAbuseRestoresRequestBodyForDownstreamHandler(t *testing.T) {
	limiter := httpapi.NewMemoryAbuseLimiter()
	mw := httpapi.NewAuthAbuseMiddleware(httpapi.DefaultAuthAbusePolicies(), limiter, nil, nil)

	var seenEmail string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		_ = json.NewDecoder(r.Body).Decode(&payload)
		seenEmail = payload["email"]
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"reader@example.com","password":"secret"}`))
	req.RemoteAddr = "192.0.2.40:1234"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if seenEmail != "reader@example.com" {
		t.Fatalf("downstream handler lost body, email=%q", seenEmail)
	}
}

func TestAuthAbuseLimiterInterface(t *testing.T) {
	var _ httpapi.AbuseLimiter = httpapi.NewMemoryAbuseLimiter()
}

func TestAuthAbuseObserveMetricsOnThrottle(t *testing.T) {
	policies := httpapi.DefaultAuthAbusePolicies()
	policies["POST /login"].Limits[0].Limit = 1

	limiter := httpapi.NewMemoryAbuseLimiter()
	registry := metrics.NewRegistry()
	mw := httpapi.NewAuthAbuseMiddleware(policies, limiter, registry, nil)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"a@example.com","password":"secret"}`))
		req.RemoteAddr = "192.0.2.50:1234"
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	rec := httptest.NewRecorder()
	registry.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "synaudio_auth_throttled_total") {
		t.Fatalf("expected auth throttle metric, got:\n%s", body)
	}
}

