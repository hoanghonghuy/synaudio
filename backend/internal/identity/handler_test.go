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
		if c.Name == identity.RefreshCookieName {
			found = true
			if !c.HttpOnly {
				t.Fatal("refresh cookie must be HttpOnly")
			}
		}
	}
	if !found {
		t.Fatalf("expected cookie %q, got %v", identity.RefreshCookieName, cookies)
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
