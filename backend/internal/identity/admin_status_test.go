package identity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func (s *fakeStore) SetUserStatusSafely(_ context.Context, targetID, status string) error {
	targetEmail := ""
	target := identity.User{}
	for email, user := range s.users {
		if user.ID == targetID {
			targetEmail = email
			target = user
			break
		}
	}
	if targetEmail == "" {
		return identity.ErrUserNotFound
	}

	if target.Status == identity.StatusActive && status != identity.StatusActive && isMfaCapableActiveAdmin(s, targetID) {
		if countMfaCapableActiveAdmins(s) <= 1 {
			return identity.ErrLastAdmin
		}
	}

	target.Status = status
	s.users[targetEmail] = target
	return nil
}

func TestAdminStatusChangeRejectsLastActiveAdminSuspension(t *testing.T) {
	store := newFakeStore()
	actor := identity.User{ID: "admin-1", Email: "admin@example.com", Status: identity.StatusActive}
	store.users[actor.Email] = actor
	store.userRoles[actor.ID] = []string{identity.RoleAdmin}
	store.rolePermissions[identity.RoleAdmin] = []string{identity.PermAdminStatusManage}
	store.mfaMethods[actor.ID] = &identity.MFAMethod{Secret: "secret", Confirmed: true}

	svc := identity.NewAuthService(store)
	err := svc.SetUserStatusAsAdmin(context.Background(), actor.ID, actor.ID, identity.StatusSuspended)
	if !errors.Is(err, identity.ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
	if got := store.users[actor.Email].Status; got != identity.StatusActive {
		t.Fatalf("last admin status changed despite guard: %s", got)
	}
}

func TestAdminStatusChangeAllowsSuspendingOneOfTwoActiveAdmins(t *testing.T) {
	store := newFakeStore()
	actor := identity.User{ID: "admin-1", Email: "admin1@example.com", Status: identity.StatusActive}
	target := identity.User{ID: "admin-2", Email: "admin2@example.com", Status: identity.StatusActive}
	store.users[actor.Email] = actor
	store.users[target.Email] = target
	store.userRoles[actor.ID] = []string{identity.RoleAdmin}
	store.userRoles[target.ID] = []string{identity.RoleAdmin}
	store.rolePermissions[identity.RoleAdmin] = []string{identity.PermAdminStatusManage}
	store.mfaMethods[actor.ID] = &identity.MFAMethod{Secret: "secret-1", Confirmed: true}
	store.mfaMethods[target.ID] = &identity.MFAMethod{Secret: "secret-2", Confirmed: true}

	svc := identity.NewAuthService(store)
	if err := svc.SetUserStatusAsAdmin(context.Background(), actor.ID, target.ID, identity.StatusSuspended); err != nil {
		t.Fatalf("suspend second admin: %v", err)
	}
	if got := store.users[target.Email].Status; got != identity.StatusSuspended {
		t.Fatalf("expected suspended target, got %s", got)
	}
}

func TestAdminStatusChangeRequiresConcretePermission(t *testing.T) {
	store := newFakeStore()
	actor := identity.User{ID: "admin-1", Email: "admin@example.com", Status: identity.StatusActive}
	target := identity.User{ID: "user-1", Email: "user@example.com", Status: identity.StatusActive}
	store.users[actor.Email] = actor
	store.users[target.Email] = target
	store.userRoles[actor.ID] = []string{identity.RoleAdmin}

	svc := identity.NewAuthService(store)
	err := svc.SetUserStatusAsAdmin(context.Background(), actor.ID, target.ID, identity.StatusSuspended)
	if !errors.Is(err, identity.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
