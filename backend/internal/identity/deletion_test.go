package identity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func TestRequestAccountDeletionDeactivatesImmediately(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, _ := svc.Register(context.Background(), "user@example.com", "password123")

	if err := svc.RequestAccountDeletion(context.Background(), u.ID); err != nil {
		t.Fatalf("request deletion: %v", err)
	}

	got, err := store.GetUserByID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.Status != identity.StatusDeactivated {
		t.Fatalf("expected DEACTIVATED, got %q", got.Status)
	}
}

func TestRequestAccountDeletionRejectsUnknownUser(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	if err := svc.RequestAccountDeletion(context.Background(), "missing"); !errors.Is(err, identity.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestCancelAccountDeletionRestoresActive(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, _ := svc.Register(context.Background(), "user@example.com", "password123")
	_ = svc.RequestAccountDeletion(context.Background(), u.ID)

	if err := svc.CancelAccountDeletion(context.Background(), u.ID); err != nil {
		t.Fatalf("cancel deletion: %v", err)
	}

	got, _ := store.GetUserByID(context.Background(), u.ID)
	if got.Status != identity.StatusActive {
		t.Fatalf("expected ACTIVE, got %q", got.Status)
	}
}

func TestPurgeAccountAnonymizesPII(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, _ := svc.Register(context.Background(), "user@example.com", "password123")
	_ = svc.RequestAccountDeletion(context.Background(), u.ID)

	if err := svc.PurgeAccount(context.Background(), u.ID); err != nil {
		t.Fatalf("purge account: %v", err)
	}

	got, _ := store.GetUserByID(context.Background(), u.ID)
	if got.Email == "user@example.com" {
		t.Fatalf("expected email anonymized, got %q", got.Email)
	}
	if got.PasswordHash != "" {
		t.Fatal("expected password hash removed")
	}
}
