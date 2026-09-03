package identity_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

type testAuthSession struct {
	cookie      *http.Cookie
	accessToken string
}

func TestMFASetupHandlerReturnsSecret(t *testing.T) {
	h := newTestHandler()
	registerUser(t, h, "user@example.com")
	sess := loginUser(t, h, "user@example.com")

	rec := doAuthenticatedJSON(t, h, http.MethodPost, "/api/v1/auth/mfa/totp/setup", nil, sess)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["secret"] == "" {
		t.Fatal("expected secret in response")
	}
}

func TestMFAConfirmHandlerReturnsRecoveryCodes(t *testing.T) {
	h := newTestHandler()
	registerUser(t, h, "user@example.com")
	sess := loginUser(t, h, "user@example.com")

	rec := doAuthenticatedJSON(t, h, http.MethodPost, "/api/v1/auth/mfa/totp/setup", nil, sess)
	var setupResp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &setupResp)
	secret, _ := setupResp["secret"].(string)

	now := identity.TOTPTimeStep(0)
	code, _ := identity.TOTPCode(secret, now)

	rec = doAuthenticatedJSON(t, h, http.MethodPost, "/api/v1/auth/mfa/totp/confirm", map[string]any{
		"code": code,
	}, sess)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	codes, ok := resp["recovery_codes"].([]any)
	if !ok || len(codes) == 0 {
		t.Fatalf("expected recovery_codes array, got %v", resp["recovery_codes"])
	}
}

func TestMFAConfirmHandlerRejectsWrongCode(t *testing.T) {
	h := newTestHandler()
	registerUser(t, h, "user@example.com")
	sess := loginUser(t, h, "user@example.com")
	doAuthenticatedJSON(t, h, http.MethodPost, "/api/v1/auth/mfa/totp/setup", nil, sess)

	rec := doAuthenticatedJSON(t, h, http.MethodPost, "/api/v1/auth/mfa/totp/confirm", map[string]any{
		"code": "000000",
	}, sess)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 400/401, got %d", rec.Code)
	}
}

func TestMFADisableHandlerReturnsOK(t *testing.T) {
	h := newTestHandler()
	registerUser(t, h, "user@example.com")
	sess := loginUser(t, h, "user@example.com")
	doAuthenticatedJSON(t, h, http.MethodPost, "/api/v1/auth/mfa/totp/setup", nil, sess)

	rec := doAuthenticatedJSON(t, h, http.MethodPost, "/api/v1/auth/mfa/totp/disable", nil, sess)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestMFAEndpointsRequireBearerAccessToken(t *testing.T) {
	h := newTestHandler()

	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/mfa/totp/setup", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected setup without access token to return 401, got %d", rec.Code)
	}
}

func doAuthenticatedJSON(t *testing.T, h http.Handler, method, path string, body any, sess testAuthSession) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+sess.accessToken)
	if sess.cookie != nil {
		req.AddCookie(sess.cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func loginUser(t *testing.T, h http.Handler, email string) testAuthSession {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": email, "password": "correct password",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login: expected refresh cookie")
	}
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Fatal("login: expected access token")
	}
	return testAuthSession{cookie: cookies[0], accessToken: resp.AccessToken}
}

func registerUser(t *testing.T, h http.Handler, email string) string {
	t.Helper()
	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": email, "password": "correct password",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	id, _ := resp["id"].(string)
	if id == "" {
		t.Fatal("expected id in register response")
	}
	return id
}
