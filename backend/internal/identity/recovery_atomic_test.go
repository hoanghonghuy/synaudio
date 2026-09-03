package identity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

type recoveryStore struct {
	*fakeStore
	failReset bool
}

func newRecoveryStore() *recoveryStore {
	return &recoveryStore{fakeStore: newFakeStore()}
}

func (s *recoveryStore) ReplaceVerificationToken(_ context.Context, userID, tokenHash string) error {
	s.verificationTokens[userID] = tokenHash
	return nil
}

func (s *recoveryStore) ReplaceResetToken(_ context.Context, userID, tokenHash string) error {
	s.resetTokens[userID] = tokenHash
	return nil
}

func (s *recoveryStore) ConsumeVerificationToken(_ context.Context, userID, tokenHash string) error {
	if s.verificationTokens[userID] != tokenHash {
		return identity.ErrInvalidToken
	}
	delete(s.verificationTokens, userID)
	return s.MarkEmailVerified(context.Background(), userID)
}

func (s *recoveryStore) ResetPasswordAtomic(_ context.Context, userID, tokenHash, passwordHash string) error {
	if s.resetTokens[userID] != tokenHash {
		return identity.ErrInvalidToken
	}
	if s.failReset {
		return errors.New("injected recovery transaction failure")
	}
	delete(s.resetTokens, userID)
	if err := s.UpdatePassword(context.Background(), userID, passwordHash); err != nil {
		return err
	}
	return s.RevokeSessions(context.Background(), userID)
}

func TestVerificationCredentialIsSingleUse(t *testing.T) {
	store := newRecoveryStore()
	svc := identity.NewAuthService(store)
	u, err := svc.Register(context.Background(), "verify@example.com", "correct password")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	token, err := svc.RequestEmailVerification(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("request verification: %v", err)
	}
	if err := svc.VerifyEmail(context.Background(), u.ID, token); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := svc.VerifyEmail(context.Background(), u.ID, token); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("replay error = %v, want ErrInvalidToken", err)
	}
}

func TestVerificationResendSupersedesOlderCredential(t *testing.T) {
	store := newRecoveryStore()
	svc := identity.NewAuthService(store)
	u, _ := svc.Register(context.Background(), "resend@example.com", "correct password")
	oldToken, _ := svc.RequestEmailVerification(context.Background(), u.ID)
	newToken, _ := svc.RequestEmailVerification(context.Background(), u.ID)

	if err := svc.VerifyEmail(context.Background(), u.ID, oldToken); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("old credential error = %v, want ErrInvalidToken", err)
	}
	if err := svc.VerifyEmail(context.Background(), u.ID, newToken); err != nil {
		t.Fatalf("new credential should verify: %v", err)
	}
}

func TestPasswordResetCredentialIsSingleUseAndRevokesSessions(t *testing.T) {
	store := newRecoveryStore()
	svc := identity.NewAuthService(store)
	u, _ := svc.Register(context.Background(), "reset@example.com", "old password")
	token, _ := svc.RequestPasswordReset(context.Background(), u.ID)

	if err := svc.ResetPassword(context.Background(), u.ID, token, "new password"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if err := svc.ResetPassword(context.Background(), u.ID, token, "other password"); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("replay error = %v, want ErrInvalidToken", err)
	}
	got, _ := store.GetUserByID(context.Background(), u.ID)
	if !identity.VerifyPassword(got.PasswordHash, "new password") {
		t.Fatal("expected first reset password to remain authoritative")
	}
	if !store.sessionsRevoked {
		t.Fatal("expected sessions revoked in recovery transaction")
	}
}

func TestPasswordResetTransactionFailurePreservesCredentialPasswordAndSessions(t *testing.T) {
	store := newRecoveryStore()
	svc := identity.NewAuthService(store)
	u, _ := svc.Register(context.Background(), "rollback@example.com", "old password")
	token, _ := svc.RequestPasswordReset(context.Background(), u.ID)
	storedHash := store.resetTokens[u.ID]
	store.failReset = true

	if err := svc.ResetPassword(context.Background(), u.ID, token, "new password"); err == nil {
		t.Fatal("expected injected transaction failure")
	}
	got, _ := store.GetUserByID(context.Background(), u.ID)
	if !identity.VerifyPassword(got.PasswordHash, "old password") {
		t.Fatal("password changed despite failed atomic recovery")
	}
	if store.sessionsRevoked {
		t.Fatal("sessions revoked despite failed atomic recovery")
	}
	if store.resetTokens[u.ID] != storedHash {
		t.Fatal("reset credential was consumed despite failed atomic recovery")
	}
}
