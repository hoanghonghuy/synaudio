package identity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func TestRevokeAdminDoesNotTreatNonAdminTargetAsLastAdmin(t *testing.T) {
	store := newFakeStore()
	actor := identity.User{ID: "admin-1", Email: "admin@example.com", Status: identity.StatusActive}
	target := identity.User{ID: "user-1", Email: "user@example.com", Status: identity.StatusActive}
	store.users[actor.Email] = actor
	store.users[target.Email] = target
	store.userRoles[actor.ID] = []string{identity.RoleAdmin}
	store.rolePermissions[identity.RoleAdmin] = []string{identity.PermAdminRoleRevoke}

	svc := identity.NewAuthService(store)
	if err := svc.RevokeAdmin(context.Background(), actor.ID, target.ID); err != nil {
		t.Fatalf("revoke non-admin target should be idempotent, got %v", err)
	}
}

func TestRevokeAdminStillRejectsLastActiveAdmin(t *testing.T) {
	store := newFakeStore()
	actor := identity.User{ID: "admin-1", Email: "admin@example.com", Status: identity.StatusActive}
	store.users[actor.Email] = actor
	store.userRoles[actor.ID] = []string{identity.RoleAdmin}
	store.rolePermissions[identity.RoleAdmin] = []string{identity.PermAdminRoleRevoke}
	store.mfaMethods[actor.ID] = &identity.MFAMethod{Secret: "secret", Confirmed: true}

	svc := identity.NewAuthService(store)
	err := svc.RevokeAdmin(context.Background(), actor.ID, actor.ID)
	if !errors.Is(err, identity.ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
}

func TestDisableTOTPRejectsLastActiveAdmin(t *testing.T) {
	store := newFakeStore()
	admin := identity.User{ID: "admin-1", Email: "admin@example.com", Status: identity.StatusActive}
	store.users[admin.Email] = admin
	store.userRoles[admin.ID] = []string{identity.RoleAdmin}
	store.mfaMethods[admin.ID] = &identity.MFAMethod{Secret: "secret", Confirmed: true}

	svc := identity.NewAuthService(store)
	err := svc.DisableTOTP(context.Background(), admin.ID)
	if !errors.Is(err, identity.ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
	if store.mfaMethods[admin.ID].Disabled {
		t.Fatal("last active admin MFA must remain enabled")
	}
}

func TestDisableTOTPRejectsWhenOnlyOtherAdminHasNoMFA(t *testing.T) {
	store := newFakeStore()
	first := identity.User{ID: "admin-1", Email: "admin1@example.com", Status: identity.StatusActive}
	second := identity.User{ID: "admin-2", Email: "admin2@example.com", Status: identity.StatusActive}
	store.users[first.Email] = first
	store.users[second.Email] = second
	store.userRoles[first.ID] = []string{identity.RoleAdmin}
	store.userRoles[second.ID] = []string{identity.RoleAdmin}
	store.mfaMethods[first.ID] = &identity.MFAMethod{Secret: "secret", Confirmed: true}

	svc := identity.NewAuthService(store)
	err := svc.DisableTOTP(context.Background(), first.ID)
	if !errors.Is(err, identity.ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin when other admin lacks MFA, got %v", err)
	}
	if store.mfaMethods[first.ID].Disabled {
		t.Fatal("sole MFA-capable admin must retain MFA")
	}
}

func TestDisableTOTPAllowsWhenAnotherMfaCapableAdminExists(t *testing.T) {
	store := newFakeStore()
	first := identity.User{ID: "admin-1", Email: "admin1@example.com", Status: identity.StatusActive}
	second := identity.User{ID: "admin-2", Email: "admin2@example.com", Status: identity.StatusActive}
	store.users[first.Email] = first
	store.users[second.Email] = second
	store.userRoles[first.ID] = []string{identity.RoleAdmin}
	store.userRoles[second.ID] = []string{identity.RoleAdmin}
	store.mfaMethods[first.ID] = &identity.MFAMethod{Secret: "secret-1", Confirmed: true}
	store.mfaMethods[second.ID] = &identity.MFAMethod{Secret: "secret-2", Confirmed: true}

	svc := identity.NewAuthService(store)
	if err := svc.DisableTOTP(context.Background(), first.ID); err != nil {
		t.Fatalf("disable MFA with another MFA-capable admin should succeed, got %v", err)
	}
	if !store.mfaMethods[first.ID].Disabled {
		t.Fatal("expected target MFA to be disabled")
	}
}

func TestAdminStatusChangeRejectsWhenOnlyOtherAdminHasNoMFA(t *testing.T) {
	store := newFakeStore()
	first := identity.User{ID: "admin-1", Email: "admin1@example.com", Status: identity.StatusActive}
	second := identity.User{ID: "admin-2", Email: "admin2@example.com", Status: identity.StatusActive}
	store.users[first.Email] = first
	store.users[second.Email] = second
	store.userRoles[first.ID] = []string{identity.RoleAdmin}
	store.userRoles[second.ID] = []string{identity.RoleAdmin}
	store.rolePermissions[identity.RoleAdmin] = []string{identity.PermAdminStatusManage}
	store.mfaMethods[first.ID] = &identity.MFAMethod{Secret: "secret-1", Confirmed: true}

	svc := identity.NewAuthService(store)
	err := svc.SetUserStatusAsAdmin(context.Background(), second.ID, first.ID, identity.StatusSuspended)
	if !errors.Is(err, identity.ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
	if got := store.users[first.Email].Status; got != identity.StatusActive {
		t.Fatalf("sole MFA-capable admin must remain ACTIVE, got %s", got)
	}
}
