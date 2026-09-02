package identity_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/synaudio/synaudio/backend/internal/identity"
)

func newTestHandler() http.Handler {
	store := newFakeStore()
	svc := identity.NewAuthService(store)
	r := chi.NewRouter()
	r.Mount("/api/v1/auth", identity.NewAuthHandler(svc))
	return r
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRegisterHandlerReturnsCreated(t *testing.T) {
	h := newTestHandler()
	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "  User@Example.COM ",
		"password": "correct horse battery staple",
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["email"] != "user@example.com" {
		t.Fatalf("expected normalized email, got %v", resp["email"])
	}
	if _, ok := resp["password_hash"]; ok {
		t.Fatal("response must not expose password hash")
	}
}

func TestRegisterHandlerRejectsDuplicate(t *testing.T) {
	h := newTestHandler()
	doJSON(t, h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "a@example.com", "password": "password123",
	})
	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "A@Example.com", "password": "password456",
	})

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestLoginHandlerSetsRefreshCookie(t *testing.T) {
	h := newTestHandler()
	doJSON(t, h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "user@example.com", "password": "correct password",
	})

	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "user@example.com", "password": "correct password",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected refresh cookie to be set")
	}
	var found bool
	for _, c := range cookies {
		if c.Name == identity.DevelopmentRefreshCookieName {
			found = true
			if !c.HttpOnly {
				t.Fatal("refresh cookie must be HttpOnly")
			}
		}
	}
	if !found {
		t.Fatalf("expected cookie %q, got %v", identity.DevelopmentRefreshCookieName, cookies)
	}
}

func TestHTTPLoginUsesBrowserCompatibleRefreshCookie(t *testing.T) {
	h := newTestHandler()
	doJSON(t, h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "http-cookie@example.com", "password": "correct password",
	})

	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "http-cookie@example.com", "password": "correct password",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected refresh cookie to be set")
	}
	cookie := cookies[0]
	if cookie.Name != identity.DevelopmentRefreshCookieName {
		t.Fatalf("expected development cookie %q, got %q", identity.DevelopmentRefreshCookieName, cookie.Name)
	}
	if cookie.Secure {
		t.Fatal("development HTTP cookie must not require Secure transport")
	}
}

func TestHTTPSLoginUsesSecureHostRefreshCookie(t *testing.T) {
	h := newTestHandler()
	doJSON(t, h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "https-cookie@example.com", "password": "correct password",
	})

	req := httptest.NewRequest(http.MethodPost, "https://example.com/api/v1/auth/login", bytes.NewBufferString(`{"email":"https-cookie@example.com","password":"correct password"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected refresh cookie to be set")
	}
	cookie := cookies[0]
	if cookie.Name != identity.RefreshCookieName {
		t.Fatalf("expected secure cookie %q, got %q", identity.RefreshCookieName, cookie.Name)
	}
	if !cookie.Secure {
		t.Fatal("HTTPS refresh cookie must require Secure transport")
	}
}

func TestCurrentUserHandlerReturnsAuthenticatedUser(t *testing.T) {
	h := newTestHandler()
	doJSON(t, h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "current@example.com", "password": "correct password",
	})

	login := doJSON(t, h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "current@example.com", "password": "correct password",
	})
	cookies := login.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected login cookie")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(cookies[0])
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := resp["email"]; got != "current@example.com" {
		t.Fatalf("expected current user email, got %v", got)
	}
}

func TestLogoutHandlerRevokesAuthenticatedSession(t *testing.T) {
	h := newTestHandler()
	doJSON(t, h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "logout@example.com", "password": "correct password",
	})

	login := doJSON(t, h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "logout@example.com", "password": "correct password",
	})
	cookies := login.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected login cookie")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(cookies[0])
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(cookies[0])
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked session to return 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRefreshHandlerRotatesRefreshCookie(t *testing.T) {
	h := newTestHandler()
	registerUser(t, h, "refresh@example.com")
	cookie := loginUser(t, h, "refresh@example.com")

	rec := doAuthenticatedJSON(t, h, http.MethodPost, "/api/v1/auth/refresh", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	refreshedCookies := rec.Result().Cookies()
	if len(refreshedCookies) == 0 || refreshedCookies[0].Value == cookie.Value {
		t.Fatal("expected refresh token rotation")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(refreshedCookies[0])
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected rotated session to remain valid, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected old session to be revoked, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRefreshHandlerRequiresSession(t *testing.T) {
	h := newTestHandler()

	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/refresh", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAccountDeletionRequestUsesAuthenticatedSession(t *testing.T) {
	h := newTestHandler()
	registerUser(t, h, "delete@example.com")
	cookie := loginUser(t, h, "delete@example.com")

	rec := doAuthenticatedJSON(t, h, http.MethodPost, "/api/v1/auth/account/deletion/request", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAccountDeletionRequestRequiresSession(t *testing.T) {
	h := newTestHandler()

	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/account/deletion/request", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAccountDeletionCancelUsesExistingSession(t *testing.T) {
	h := newTestHandler()
	registerUser(t, h, "cancel-delete@example.com")
	cookie := loginUser(t, h, "cancel-delete@example.com")

	rec := doAuthenticatedJSON(t, h, http.MethodPost, "/api/v1/auth/account/deletion/request", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("request deletion: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doAuthenticatedJSON(t, h, http.MethodPost, "/api/v1/auth/account/deletion/cancel", nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("cancel deletion: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected restored account session to remain valid, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLoginHandlerRejectsWrongPassword(t *testing.T) {
	h := newTestHandler()
	doJSON(t, h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "user@example.com", "password": "correct password",
	})

	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "user@example.com", "password": "wrong password",
	})

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

var _ = context.Background
