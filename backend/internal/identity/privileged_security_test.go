package identity_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

type privilegedSecurityFakeStore struct {
	*fakeStore
	confirmAtomicErr error
}

func newPrivilegedSecurityFakeStore() *privilegedSecurityFakeStore {
	base := newFakeStore()
	base.recoveryHashes = map[string]bool{}
	base.assuredSessions = map[string]time.Time{}
	base.recentSessions = map[string]time.Time{}
	return &privilegedSecurityFakeStore{
		fakeStore: base,
	}
}

func (s *privilegedSecurityFakeStore) ConfirmMFAWithRecoveryCodes(ctx context.Context, userID string, hashes []string) error {
	if s.confirmAtomicErr != nil {
		return s.confirmAtomicErr
	}
	if err := s.fakeStore.ConfirmMFAMethod(ctx, userID); err != nil {
		return err
	}
	return s.ReplaceMFARecoveryCodes(ctx, userID, hashes)
}

func TestAdminRoleAloneDoesNotGrantPrivilegedCapability(t *testing.T) {
	store := newPrivilegedSecurityFakeStore()
	svc := identity.NewAuthService(store)
	u, err := svc.Register(context.Background(), "admin@example.com", "correct password")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.GrantRole(context.Background(), u.ID, identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	sess, err := svc.Login(context.Background(), u.Email, "correct password")
	if err != nil {
		t.Fatal(err)
	}
	access, err := svc.IssueAccessToken(sess)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/api/v1/admin/audit", nil)
	req.Header.Set("Authorization", "Bearer "+access.Token)

	allowed, err := svc.ResolveAdmin(context.Background(), req)
	if err == nil || !errors.Is(err, identity.ErrEmailVerificationRequired) {
		t.Fatalf("expected ErrEmailVerificationRequired, got allowed=%v err=%v", allowed, err)
	}
	if allowed {
		t.Fatal("ADMIN role without verified email/MFA session assurance must be denied")
	}
}

func TestConfirmTOTPDoesNotPartiallyRotateRecoveryCodesOnAtomicFailure(t *testing.T) {
	store := newPrivilegedSecurityFakeStore()
	svc := identity.NewAuthService(store)
	u, err := svc.Register(context.Background(), "admin@example.com", "correct password")
	if err != nil {
		t.Fatal(err)
	}
	secret, err := svc.SetupTOTP(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	store.recoveryHashes["existing-hash"] = true
	store.confirmAtomicErr = errors.New("confirm write failed")
	counter := identity.TOTPTimeStep(0)
	code, err := identity.TOTPCode(secret, counter)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.ConfirmTOTP(context.Background(), u.ID, code, counter); err == nil {
		t.Fatal("confirmation failure must be returned")
	}
	if len(store.recoveryHashes) != 1 || !store.recoveryHashes["existing-hash"] {
		t.Fatal("failed atomic confirmation must preserve the prior recovery-code set")
	}
	method, err := store.GetMFAMethod(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if method.Confirmed {
		t.Fatal("failed atomic confirmation must not confirm MFA")
	}
}

func TestVerifiedAdminRequiresExactSessionMFAAssurance(t *testing.T) {
	store := newPrivilegedSecurityFakeStore()
	svc := identity.NewAuthService(store)
	u, err := svc.Register(context.Background(), "admin@example.com", "correct password")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MarkEmailVerified(context.Background(), u.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.GrantRole(context.Background(), u.ID, identity.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	secret, err := svc.SetupTOTP(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	counter := identity.TOTPTimeStep(0)
	code, err := identity.TOTPCode(secret, counter)
	if err != nil {
		t.Fatal(err)
	}
	codes, err := svc.ConfirmTOTP(context.Background(), u.ID, code, counter)
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) == 0 || len(store.recoveryHashes) != len(codes) {
		t.Fatal("recovery codes must be persisted as hashes")
	}

	first, err := svc.Login(context.Background(), u.Email, "correct password")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Login(context.Background(), u.Email, "correct password")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkSessionMFAAndRecentAuth(context.Background(), identity.Principal{UserID: u.ID, SessionID: first.ID}); err != nil {
		t.Fatal(err)
	}

	firstAccess, _ := svc.IssueAccessToken(first)
	firstReq := httptest.NewRequest("GET", "/api/v1/admin/audit", nil)
	firstReq.Header.Set("Authorization", "Bearer "+firstAccess.Token)
	allowed, err := svc.ResolveAdmin(context.Background(), firstReq)
	if err != nil || !allowed {
		t.Fatalf("MFA-assured verified admin session must be allowed: allowed=%v err=%v", allowed, err)
	}

	secondAccess, _ := svc.IssueAccessToken(second)
	secondReq := httptest.NewRequest("GET", "/api/v1/admin/audit", nil)
	secondReq.Header.Set("Authorization", "Bearer "+secondAccess.Token)
	allowed, err = svc.ResolveAdmin(context.Background(), secondReq)
	if err == nil || !errors.Is(err, identity.ErrMFARequired) {
		t.Fatalf("expected ErrMFARequired for parallel session, got allowed=%v err=%v", allowed, err)
	}
	if allowed {
		t.Fatal("MFA assurance from one session must not leak to a parallel password-only session")
	}

	consumed, err := svc.ConsumeRecoveryCode(context.Background(), u.ID, codes[0])
	if err != nil || !consumed {
		t.Fatalf("first recovery-code use must succeed: consumed=%v err=%v", consumed, err)
	}
	consumed, err = svc.ConsumeRecoveryCode(context.Background(), u.ID, codes[0])
	if err != nil {
		t.Fatal(err)
	}
	if consumed {
		t.Fatal("recovery code must be single-use")
	}
}

func TestReAuthWithTOTPGrantsPrivilegedCapability(t *testing.T) {
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
	principal := identity.Principal{UserID: u.ID, SessionID: sess.ID}
	totpCode, _ := identity.TOTPCode(secret, counter)
	if err := svc.VerifyPrivilegedMFAChallenge(context.Background(), principal, totpCode, ""); err != nil {
		t.Fatalf("re-auth with TOTP: %v", err)
	}

	access, _ := svc.IssueAccessToken(sess)
	req := httptest.NewRequest("GET", "/api/v1/admin/audit", nil)
	req.Header.Set("Authorization", "Bearer "+access.Token)
	allowed, err := svc.ResolveAdmin(context.Background(), req)
	if err != nil || !allowed {
		t.Fatalf("TOTP re-auth must grant privileged capability: allowed=%v err=%v", allowed, err)
	}
}

func TestRevokedAdminRoleDeniedImmediately(t *testing.T) {
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

	_ = store.RevokeRole(context.Background(), u.ID, identity.RoleAdmin)
	req := httptest.NewRequest("GET", "/api/v1/admin/audit", nil)
	req.Header.Set("Authorization", "Bearer "+access.Token)
	allowed, err := svc.ResolveAdmin(context.Background(), req)
	if err == nil || !errors.Is(err, identity.ErrForbidden) {
		t.Fatalf("revoked admin must be denied immediately: allowed=%v err=%v", allowed, err)
	}
}

func TestSuspendedAccountDeniedAtPrivilegedBoundary(t *testing.T) {
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

	u.Status = identity.StatusSuspended
	store.users[u.Email] = u
	req := httptest.NewRequest("GET", "/api/v1/admin/audit", nil)
	req.Header.Set("Authorization", "Bearer "+access.Token)
	allowed, err := svc.ResolveAdmin(context.Background(), req)
	if err == nil || !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatalf("suspended account must lose privileged capability: allowed=%v err=%v", allowed, err)
	}
}

func TestReAuthWithRecoveryCodeGrantsPrivilegedCapability(t *testing.T) {
	store := newPrivilegedSecurityFakeStore()
	svc := identity.NewAuthService(store)
	u, _ := svc.Register(context.Background(), "admin@example.com", "correct password")
	_ = store.MarkEmailVerified(context.Background(), u.ID)
	_ = store.GrantRole(context.Background(), u.ID, identity.RoleAdmin)
	secret, _ := svc.SetupTOTP(context.Background(), u.ID)
	counter := identity.TOTPTimeStep(0)
	code, _ := identity.TOTPCode(secret, counter)
	recoveryCodes, _ := svc.ConfirmTOTP(context.Background(), u.ID, code, counter)

	sess, _ := svc.Login(context.Background(), u.Email, "correct password")
	principal := identity.Principal{UserID: u.ID, SessionID: sess.ID}
	if err := svc.VerifyPrivilegedMFAChallenge(context.Background(), principal, "", recoveryCodes[0]); err != nil {
		t.Fatalf("re-auth with recovery code: %v", err)
	}

	access, _ := svc.IssueAccessToken(sess)
	req := httptest.NewRequest("GET", "/api/v1/admin/audit", nil)
	req.Header.Set("Authorization", "Bearer "+access.Token)
	allowed, err := svc.ResolveAdmin(context.Background(), req)
	if err != nil || !allowed {
		t.Fatalf("recovery re-auth must grant privileged capability: allowed=%v err=%v", allowed, err)
	}
}
