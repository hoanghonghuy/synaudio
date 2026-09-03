package identity_test

import (
	"bytes"
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
		"email": "  User@Example.COM ", "password": "correct horse battery staple",
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
	registerUser(t, h, "a@example.com")
	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "A@Example.com", "password": "password456",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestLoginReturnsAccessTokenAndHttpOnlyRefreshCookie(t *testing.T) {
	h := newTestHandler()
	registerUser(t, h, "login@example.com")

	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "login@example.com", "password": "correct password",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["access_token"] == "" || resp["token_type"] != "Bearer" {
		t.Fatalf("expected bearer access token, got %v", resp)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != identity.DevelopmentRefreshCookieName || !cookies[0].HttpOnly {
		t.Fatalf("expected HttpOnly development refresh cookie, got %v", cookies)
	}
}

func TestHTTPSLoginUsesSecureHostRefreshCookie(t *testing.T) {
	h := newTestHandler()
	registerUser(t, h, "https-cookie@example.com")

	req := httptest.NewRequest(http.MethodPost, "https://example.com/api/v1/auth/login", bytes.NewBufferString(`{"email":"https-cookie@example.com","password":"correct password"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != identity.RefreshCookieName || !cookies[0].Secure {
		t.Fatalf("expected secure __Host refresh cookie, got %v", cookies)
	}
}

func TestCurrentUserRequiresBearerNotRefreshCookie(t *testing.T) {
	h := newTestHandler()
	registerUser(t, h, "current@example.com")
	sess := loginUser(t, h, "current@example.com")

	cookieOnly := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	cookieOnly.AddCookie(sess.cookie)
	cookieRec := httptest.NewRecorder()
	h.ServeHTTP(cookieRec, cookieOnly)
	if cookieRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected refresh-cookie-only request to be 401, got %d", cookieRec.Code)
	}

	rec := doAuthenticatedJSON(t, h, http.MethodGet, "/api/v1/auth/me", nil, sess)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected bearer access to return 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRefreshRotatesCredentialAndRejectsReuse(t *testing.T) {
	h := newTestHandler()
	registerUser(t, h, "refresh@example.com")
	initial := loginUser(t, h, "refresh@example.com")

	first := refreshWithCookie(t, h, initial.cookie)
	if first.Code != http.StatusOK {
		t.Fatalf("refresh: expected 200, got %d: %s", first.Code, first.Body.String())
	}
	cookies := first.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Value == initial.cookie.Value {
		t.Fatal("expected rotated refresh credential")
	}
	var refreshed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &refreshed); err != nil || refreshed.AccessToken == "" {
		t.Fatalf("expected fresh access token: %v", err)
	}

	replay := refreshWithCookie(t, h, initial.cookie)
	if replay.Code != http.StatusUnauthorized {
		t.Fatalf("expected old refresh credential reuse to be 401, got %d", replay.Code)
	}

	rotated := testAuthSession{cookie: cookies[0], accessToken: refreshed.AccessToken}
	me := doAuthenticatedJSON(t, h, http.MethodGet, "/api/v1/auth/me", nil, rotated)
	if me.Code != http.StatusOK {
		t.Fatalf("expected fresh access token to work, got %d: %s", me.Code, me.Body.String())
	}
}

func TestLogoutRevokesBoundAccessTokenImmediately(t *testing.T) {
	h := newTestHandler()
	registerUser(t, h, "logout@example.com")
	sess := loginUser(t, h, "logout@example.com")

	rec := doAuthenticatedJSON(t, h, http.MethodPost, "/api/v1/auth/logout", nil, sess)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	me := doAuthenticatedJSON(t, h, http.MethodGet, "/api/v1/auth/me", nil, sess)
	if me.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked access token to be 401, got %d", me.Code)
	}
}

func TestLogoutAllRevokesEverySession(t *testing.T) {
	h := newTestHandler()
	registerUser(t, h, "all@example.com")
	first := loginUser(t, h, "all@example.com")
	second := loginUser(t, h, "all@example.com")

	rec := doAuthenticatedJSON(t, h, http.MethodPost, "/api/v1/auth/logout-all", nil, first)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout-all: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	for _, sess := range []testAuthSession{first, second} {
		me := doAuthenticatedJSON(t, h, http.MethodGet, "/api/v1/auth/me", nil, sess)
		if me.Code != http.StatusUnauthorized {
			t.Fatalf("expected all sessions revoked, got %d", me.Code)
		}
	}
}

func TestSessionListingAndTargetedRevocation(t *testing.T) {
	h := newTestHandler()
	registerUser(t, h, "sessions@example.com")
	current := loginUser(t, h, "sessions@example.com")
	other := loginUser(t, h, "sessions@example.com")

	list := doAuthenticatedJSON(t, h, http.MethodGet, "/api/v1/auth/sessions", nil, current)
	if list.Code != http.StatusOK {
		t.Fatalf("list sessions: expected 200, got %d: %s", list.Code, list.Body.String())
	}
	var body struct {
		Items []struct {
			ID      string `json:"id"`
			Current bool   `json:"current"`
		} `json:"items"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("expected two sessions, got %d", len(body.Items))
	}
	var otherID string
	for _, item := range body.Items {
		if !item.Current {
			otherID = item.ID
		}
	}
	if otherID == "" {
		t.Fatal("expected non-current session id")
	}

	revoke := doAuthenticatedJSON(t, h, http.MethodDelete, "/api/v1/auth/sessions/"+otherID, nil, current)
	if revoke.Code != http.StatusOK {
		t.Fatalf("revoke session: expected 200, got %d: %s", revoke.Code, revoke.Body.String())
	}
	me := doAuthenticatedJSON(t, h, http.MethodGet, "/api/v1/auth/me", nil, other)
	if me.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked device access token to be 401, got %d", me.Code)
	}
}

func TestAccountDeletionRequiresAndUsesBearerSession(t *testing.T) {
	h := newTestHandler()
	unauth := doJSON(t, h, http.MethodPost, "/api/v1/auth/account/deletion/request", nil)
	if unauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated deletion request to be 401, got %d", unauth.Code)
	}

	registerUser(t, h, "delete@example.com")
	sess := loginUser(t, h, "delete@example.com")
	rec := doAuthenticatedJSON(t, h, http.MethodPost, "/api/v1/auth/account/deletion/request", nil, sess)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLoginHandlerRejectsWrongPassword(t *testing.T) {
	h := newTestHandler()
	registerUser(t, h, "wrong@example.com")
	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "wrong@example.com", "password": "wrong password",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func refreshWithCookie(t *testing.T, h http.Handler, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
