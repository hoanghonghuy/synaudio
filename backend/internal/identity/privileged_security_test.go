package identity_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

type privilegedSecurityFakeStore struct {
	*fakeStore
	recoveryHashes map[string]bool
	assuredSessions map[string]time.Time
	recentSessions map[string]time.Time
}

func newPrivilegedSecurityFakeStore() *privilegedSecurityFakeStore {
	return &privilegedSecurityFakeStore{
		fakeStore:        newFakeStore(),
		recoveryHashes:   map[string]bool{},
		assuredSessions:  map[string]time.Time{},
		recentSessions:   map[string]time.Time{},
	}
}

func (s *privilegedSecurityFakeStore) ReplaceMFARecoveryCodes(_ context.Context, _ string, hashes []string) error {
	s.recoveryHashes = map[string]bool{}
	for _, hash := range hashes {
		s.recoveryHashes[hash] = true
	}
	return nil
}

func (s *privilegedSecurityFakeStore) ConsumeMFARecoveryCode(_ context.Context, _ string, hash string) (bool, error) {
	if !s.recoveryHashes[hash] {
		return false, nil
	}
	delete(s.recoveryHashes, hash)
	return true, nil
}

func (s *privilegedSecurityFakeStore) MarkSessionMFAAndRecentAuth(_ context.Context, userID, sessionID string, at time.Time) error {
	sess, ok := s.sessions[sessionID]
	if !ok || sess.UserID != userID {
		return identity.ErrUnauthenticated
	}
	s.assuredSessions[sessionID] = at
	s.recentSessions[sessionID] = at
	return nil
}

func (s *privilegedSecurityFakeStore) HasPrivilegedSessionAssurance(_ context.Context, userID, sessionID string, _ time.Time) (bool, error) {
	sess, ok := s.sessions[sessionID]
	if !ok || sess.UserID != userID {
		return false, nil
	}
	_, ok = s.assuredSessions[sessionID]
	return ok, nil
}

func (s *privilegedSecurityFakeStore) HasRecentAuth(_ context.Context, userID, sessionID string, cutoff time.Time) (bool, error) {
	sess, ok := s.sessions[sessionID]
	if !ok || sess.UserID != userID {
		return false, nil
	}
	at, ok := s.recentSessions[sessionID]
	return ok && !at.Before(cutoff), nil
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
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Fatal("ADMIN role without verified email/MFA session assurance must be denied")
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
	if err != nil {
		t.Fatal(err)
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
