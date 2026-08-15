package identity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func TestSetupTOTPReturnsSecretAndStoresUnconfirmed(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, _ := svc.Register(context.Background(), "user@example.com", "correct password")

	secret, err := svc.SetupTOTP(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("setup totp: %v", err)
	}
	if secret == "" {
		t.Fatal("expected a secret")
	}

	method := store.mfaMethods[u.ID]
	if method == nil {
		t.Fatal("expected MFA method to be stored")
	}
	if method.Confirmed {
		t.Fatal("MFA method must be unconfirmed after setup")
	}
	if method.Secret == "" {
		t.Fatal("expected encrypted secret to be stored")
	}
}

func TestConfirmTOTPValidatesCodeAndReturnsRecoveryCodes(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, _ := svc.Register(context.Background(), "user@example.com", "correct password")
	secret, _ := svc.SetupTOTP(context.Background(), u.ID)

	now := identity.TOTPTimeStep(0)
	code, _ := identity.TOTPCode(secret, now)

	codes, err := svc.ConfirmTOTP(context.Background(), u.ID, code, now)
	if err != nil {
		t.Fatalf("confirm totp: %v", err)
	}
	if len(codes) == 0 {
		t.Fatal("expected recovery codes")
	}

	method := store.mfaMethods[u.ID]
	if !method.Confirmed {
		t.Fatal("MFA method must be confirmed")
	}
}

func TestConfirmTOTPRejectsWrongCode(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, _ := svc.Register(context.Background(), "user@example.com", "correct password")
	_, _ = svc.SetupTOTP(context.Background(), u.ID)

	now := identity.TOTPTimeStep(0)
	if _, err := svc.ConfirmTOTP(context.Background(), u.ID, "000000", now); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestDisableTOTPMarksDisabled(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, _ := svc.Register(context.Background(), "user@example.com", "correct password")
	secret, _ := svc.SetupTOTP(context.Background(), u.ID)
	now := identity.TOTPTimeStep(0)
	code, _ := identity.TOTPCode(secret, now)
	_, _ = svc.ConfirmTOTP(context.Background(), u.ID, code, now)

	if err := svc.DisableTOTP(context.Background(), u.ID); err != nil {
		t.Fatalf("disable totp: %v", err)
	}

	method := store.mfaMethods[u.ID]
	if !method.Disabled {
		t.Fatal("MFA method must be disabled")
	}
}
