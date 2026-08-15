package identity_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func TestMFASetupHandlerReturnsSecret(t *testing.T) {
	h := newTestHandler()
	doJSON(t, h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "user@example.com", "password": "correct password",
	})

	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/mfa/totp/setup", map[string]string{
		"user_id": "user-1",
	})

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
	doJSON(t, h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "user@example.com", "password": "correct password",
	})

	// Setup to obtain a valid secret.
	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/mfa/totp/setup", map[string]string{
		"user_id": "user-1",
	})
	var setupResp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &setupResp)
	secret, _ := setupResp["secret"].(string)

	now := identity.TOTPTimeStep(0)
	code, _ := identity.TOTPCode(secret, now)

	rec = doJSON(t, h, http.MethodPost, "/api/v1/auth/mfa/totp/confirm", map[string]any{
		"user_id": "user-1",
		"code":    code,
	})

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
	doJSON(t, h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "user@example.com", "password": "correct password",
	})
	doJSON(t, h, http.MethodPost, "/api/v1/auth/mfa/totp/setup", map[string]string{
		"user_id": "user-1",
	})

	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/mfa/totp/confirm", map[string]any{
		"user_id": "user-1",
		"code":    "000000",
	})

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 400/401, got %d", rec.Code)
	}
}

func TestMFADisableHandlerReturnsOK(t *testing.T) {
	h := newTestHandler()
	doJSON(t, h, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "user@example.com", "password": "correct password",
	})
	doJSON(t, h, http.MethodPost, "/api/v1/auth/mfa/totp/setup", map[string]string{
		"user_id": "user-1",
	})

	rec := doJSON(t, h, http.MethodPost, "/api/v1/auth/mfa/totp/disable", map[string]string{
		"user_id": "user-1",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
