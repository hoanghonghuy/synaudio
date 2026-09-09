package identity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func (s *fakeStore) RequestAccountDeletionSafely(ctx context.Context, userID string) error {
	u, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.Status == identity.StatusActive && isMfaCapableActiveAdmin(s, userID) {
		if countMfaCapableActiveAdmins(s) <= 1 {
			return identity.ErrLastAdmin
		}
	}
	if err := s.DeactivateUser(ctx, userID); err != nil {
		return err
	}
	return s.RevokeSessions(ctx, userID)
}

func TestAccountDeletionRejectsFinalActiveAdmin(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	admin, err := svc.Register(context.Background(), "admin@example.com", "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	store.userRoles[admin.ID] = []string{identity.RoleAdmin}
	store.mfaMethods[admin.ID] = &identity.MFAMethod{Secret: "secret", Confirmed: true}

	if err := svc.RequestAccountDeletion(context.Background(), admin.ID); !errors.Is(err, identity.ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
	got, _ := store.GetUserByID(context.Background(), admin.ID)
	if got.Status != identity.StatusActive {
		t.Fatalf("final admin must remain ACTIVE, got %q", got.Status)
	}
	if store.sessionsRevoked {
		t.Fatal("final admin sessions must not be revoked when deletion is rejected")
	}
}

func TestAccountDeletionAllowsOneOfTwoActiveAdmins(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	first, _ := svc.Register(context.Background(), "admin1@example.com", "password123")
	second, _ := svc.Register(context.Background(), "admin2@example.com", "password123")
	store.userRoles[first.ID] = []string{identity.RoleAdmin}
	store.userRoles[second.ID] = []string{identity.RoleAdmin}
	store.mfaMethods[first.ID] = &identity.MFAMethod{Secret: "secret-1", Confirmed: true}
	store.mfaMethods[second.ID] = &identity.MFAMethod{Secret: "secret-2", Confirmed: true}

	if err := svc.RequestAccountDeletion(context.Background(), first.ID); err != nil {
		t.Fatalf("request deletion: %v", err)
	}
	got, _ := store.GetUserByID(context.Background(), first.ID)
	if got.Status != identity.StatusDeactivated {
		t.Fatalf("expected DEACTIVATED, got %q", got.Status)
	}
	if !store.sessionsRevoked {
		t.Fatal("expected sessions revoked with deletion deactivation")
	}
}

func TestAccountDeletionRejectsWhenOnlyOtherAdminHasNoMFA(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	first, _ := svc.Register(context.Background(), "admin1@example.com", "password123")
	second, _ := svc.Register(context.Background(), "admin2@example.com", "password123")
	store.userRoles[first.ID] = []string{identity.RoleAdmin}
	store.userRoles[second.ID] = []string{identity.RoleAdmin}
	store.mfaMethods[first.ID] = &identity.MFAMethod{Secret: "secret-1", Confirmed: true}

	if err := svc.RequestAccountDeletion(context.Background(), first.ID); !errors.Is(err, identity.ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
	got, _ := store.GetUserByID(context.Background(), first.ID)
	if got.Status != identity.StatusActive {
		t.Fatalf("sole MFA-capable admin must remain ACTIVE, got %q", got.Status)
	}
}
