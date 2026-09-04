package identity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

// RequestAccountDeletionSafely lets the shared fakeStore model the production
// guard contract while keeping the service tests focused on behavior.
func (s *fakeStore) RequestAccountDeletionSafely(ctx context.Context, userID string) error {
	u, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.Status == identity.StatusActive && contains(s.userRoles[userID], identity.RoleAdmin) {
		activeAdmins := 0
		for id, roles := range s.userRoles {
			candidate, userErr := s.GetUserByID(ctx, id)
			if userErr == nil && candidate.Status == identity.StatusActive && contains(roles, identity.RoleAdmin) {
				activeAdmins++
			}
		}
		if activeAdmins <= 1 {
			return identity.ErrLastAdmin
		}
	}
	if err := s.DeactivateUser(ctx, userID); err != nil {
		return err
	}
	return s.RevokeSessions(ctx, userID)
}

func TestRequestAccountDeletionRejectsFinalActiveAdmin(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	admin, err := svc.Register(context.Background(), "admin@example.com", "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	store.userRoles[admin.ID] = []string{identity.RoleAdmin}

	if err := svc.RequestAccountDeletion(context.Background(), admin.ID); !errors.Is(err, identity.ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
	got, err := store.GetUserByID(context.Background(), admin.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.Status != identity.StatusActive {
		t.Fatalf("final admin must remain ACTIVE, got %q", got.Status)
	}
}

func TestRequestAccountDeletionAllowsOneOfTwoActiveAdmins(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	first, _ := svc.Register(context.Background(), "admin1@example.com", "password123")
	second, _ := svc.Register(context.Background(), "admin2@example.com", "password123")
	store.userRoles[first.ID] = []string{identity.RoleAdmin}
	store.userRoles[second.ID] = []string{identity.RoleAdmin}

	if err := svc.RequestAccountDeletion(context.Background(), first.ID); err != nil {
		t.Fatalf("request deletion: %v", err)
	}
	got, _ := store.GetUserByID(context.Background(), first.ID)
	if got.Status != identity.StatusDeactivated {
		t.Fatalf("expected DEACTIVATED, got %q", got.Status)
	}
	if !store.sessionsRevoked {
		t.Fatal("expected sessions to be revoked with deactivation")
	}
}
