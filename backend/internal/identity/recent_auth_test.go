package identity_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func TestRequireRecentAuthRejectsStaleAssurance(t *testing.T) {
	store := newPrivilegedSecurityFakeStore()
	svc := identity.NewAuthService(store, identity.WithAuthSettings(identity.AuthSettings{
		RecentAuthWindow: 10 * time.Minute,
	}))
	u, _ := svc.Register(context.Background(), "admin@example.com", "correct password")
	_ = store.MarkEmailVerified(context.Background(), u.ID)
	_ = store.GrantRole(context.Background(), u.ID, identity.RoleAdmin)
	secret, _ := svc.SetupTOTP(context.Background(), u.ID)
	counter := identity.TOTPTimeStep(0)
	code, _ := identity.TOTPCode(secret, counter)
	_, _ = svc.ConfirmTOTP(context.Background(), u.ID, code, counter)

	sess, _ := svc.Login(context.Background(), u.Email, "correct password")
	_ = svc.MarkSessionMFAAndRecentAuth(context.Background(), identity.Principal{UserID: u.ID, SessionID: sess.ID})
	store.recentSessions[sess.ID] = time.Now().UTC().Add(-11 * time.Minute)

	access, _ := svc.IssueAccessToken(sess)
	req := httptest.NewRequest("POST", "/api/v1/admin/users/target/roles/admin", nil)
	req.Header.Set("Authorization", "Bearer "+access.Token)

	if err := svc.RequireRecentAuth(context.Background(), req); !errors.Is(err, identity.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for stale recent-auth, got %v", err)
	}
}

func TestRequireSessionRecentAuthRejectsStaleAssuranceWithoutAdminRole(t *testing.T) {
	store := newPrivilegedSecurityFakeStore()
	svc := identity.NewAuthService(store, identity.WithAuthSettings(identity.AuthSettings{
		RecentAuthWindow: 10 * time.Minute,
	}))
	u, _ := svc.Register(context.Background(), "user@example.com", "correct password")
	secret, _ := svc.SetupTOTP(context.Background(), u.ID)
	counter := identity.TOTPTimeStep(0)
	code, _ := identity.TOTPCode(secret, counter)
	_, _ = svc.ConfirmTOTP(context.Background(), u.ID, code, counter)

	sess, _ := svc.Login(context.Background(), u.Email, "correct password")
	_ = svc.MarkSessionMFAAndRecentAuth(context.Background(), identity.Principal{UserID: u.ID, SessionID: sess.ID})
	store.recentSessions[sess.ID] = time.Now().UTC().Add(-11 * time.Minute)

	access, _ := svc.IssueAccessToken(sess)
	req := httptest.NewRequest("POST", "/api/v1/auth/mfa/totp/disable", nil)
	req.Header.Set("Authorization", "Bearer "+access.Token)

	if err := svc.RequireSessionRecentAuth(context.Background(), req); !errors.Is(err, identity.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for stale session recent-auth, got %v", err)
	}
}

func TestRequireSessionRecentAuthAllowsFreshAssuranceWithoutAdminRole(t *testing.T) {
	store := newPrivilegedSecurityFakeStore()
	svc := identity.NewAuthService(store)
	u, _ := svc.Register(context.Background(), "user@example.com", "correct password")
	secret, _ := svc.SetupTOTP(context.Background(), u.ID)
	counter := identity.TOTPTimeStep(0)
	code, _ := identity.TOTPCode(secret, counter)
	_, _ = svc.ConfirmTOTP(context.Background(), u.ID, code, counter)

	sess, _ := svc.Login(context.Background(), u.Email, "correct password")
	_ = svc.MarkSessionMFAAndRecentAuth(context.Background(), identity.Principal{UserID: u.ID, SessionID: sess.ID})

	access, _ := svc.IssueAccessToken(sess)
	req := httptest.NewRequest("POST", "/api/v1/auth/account/deletion/request", nil)
	req.Header.Set("Authorization", "Bearer "+access.Token)

	if err := svc.RequireSessionRecentAuth(context.Background(), req); err != nil {
		t.Fatalf("fresh session recent-auth must pass: %v", err)
	}
}

func TestRequireRecentAuthAllowsFreshAssurance(t *testing.T) {
	store := newPrivilegedSecurityFakeStore()
	svc := identity.NewAuthService(store)
	u, _ := svc.Register(context.Background(), "admin@example.com", "correct password")
	_ = store.MarkEmailVerified(context.Background(), u.ID)
	_ = store.GrantRole(context.Background(), u.ID, identity.RoleAdmin)
	secret, _ := svc.SetupTOTP(context.Background(), u.ID)
	counter := identity.TOTPTimeStep(0)
	code, _ := identity.TOTPCode(secret, counter)
	_, _ = svc.ConfirmTOTP(context.Background(), u.ID, code, counter)

	sess, _ := svc.Login(context.Background(), u.Email, "correct password")
	_ = svc.MarkSessionMFAAndRecentAuth(context.Background(), identity.Principal{UserID: u.ID, SessionID: sess.ID})

	access, _ := svc.IssueAccessToken(sess)
	req := httptest.NewRequest("POST", "/api/v1/admin/users/target/roles/admin", nil)
	req.Header.Set("Authorization", "Bearer "+access.Token)

	if err := svc.RequireRecentAuth(context.Background(), req); err != nil {
		t.Fatalf("fresh recent-auth must pass: %v", err)
	}
}
