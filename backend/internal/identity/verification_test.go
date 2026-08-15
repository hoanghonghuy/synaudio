package identity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func TestRequestEmailVerificationCreatesHashedToken(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, err := svc.Register(context.Background(), "user@example.com", "correct password")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	token, err := svc.RequestEmailVerification(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("request verification: %v", err)
	}
	if token == "" {
		t.Fatal("expected a verification token")
	}

	stored := store.verificationTokens[u.ID]
	if stored == "" {
		t.Fatal("expected token to be stored")
	}
	if stored == token {
		t.Fatal("stored token must be hashed, not raw")
	}
	if !identity.VerifyTokenHash(stored, token) {
		t.Fatal("stored hash must verify against raw token")
	}
}

func TestVerifyEmailMarksVerified(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, _ := svc.Register(context.Background(), "user@example.com", "correct password")
	token, _ := svc.RequestEmailVerification(context.Background(), u.ID)

	if err := svc.VerifyEmail(context.Background(), u.ID, token); err != nil {
		t.Fatalf("verify email: %v", err)
	}

	got := store.users[u.Email]
	if got.EmailVerifiedAt == "" {
		t.Fatal("expected email_verified_at to be set")
	}
}

func TestVerifyEmailRejectsWrongToken(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, _ := svc.Register(context.Background(), "user@example.com", "correct password")
	_, _ = svc.RequestEmailVerification(context.Background(), u.ID)

	if err := svc.VerifyEmail(context.Background(), u.ID, "wrong-token"); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestRequestPasswordResetCreatesHashedToken(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, _ := svc.Register(context.Background(), "user@example.com", "correct password")

	token, err := svc.RequestPasswordReset(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("request reset: %v", err)
	}
	if token == "" {
		t.Fatal("expected a reset token")
	}

	stored := store.resetTokens[u.ID]
	if stored == "" {
		t.Fatal("expected reset token to be stored")
	}
	if stored == token {
		t.Fatal("stored reset token must be hashed")
	}
}

func TestResetPasswordUpdatesHashAndRevokesSessions(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, _ := svc.Register(context.Background(), "user@example.com", "old password")
	token, _ := svc.RequestPasswordReset(context.Background(), u.ID)

	if err := svc.ResetPassword(context.Background(), u.ID, token, "new password"); err != nil {
		t.Fatalf("reset password: %v", err)
	}

	got := store.users[u.Email]
	if !identity.VerifyPassword(got.PasswordHash, "new password") {
		t.Fatal("expected password to be updated")
	}
	if identity.VerifyPassword(got.PasswordHash, "old password") {
		t.Fatal("old password must no longer verify")
	}
	if !store.sessionsRevoked {
		t.Fatal("expected sessions to be revoked on password reset")
	}
}

func TestResetPasswordRejectsWrongToken(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, _ := svc.Register(context.Background(), "user@example.com", "old password")
	_, _ = svc.RequestPasswordReset(context.Background(), u.ID)

	if err := svc.ResetPassword(context.Background(), u.ID, "wrong-token", "new password"); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
