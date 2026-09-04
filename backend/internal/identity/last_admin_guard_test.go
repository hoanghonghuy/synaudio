package identity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

// RevokeAdminRoleSafely equips the shared fakeStore with the same atomic
// decision semantics required from production persistence. It intentionally
// checks whether the target is itself an ACTIVE admin before applying the
// last-admin guard; revoking a non-admin must remain idempotent.
func (s *fakeStore) RevokeAdminRoleSafely(_ context.Context, targetID string) error {
	targetActiveAdmin := false
	activeAdmins := 0
	for _, u := range s.users {
		isAdmin := false
		for _, role := range s.userRoles[u.ID] {
			if role == identity.RoleAdmin {
				isAdmin = true
				break
			}
		}
		if isAdmin && u.Status == identity.StatusActive {
			activeAdmins++
			if u.ID == targetID {
				targetActiveAdmin = true
			}
		}
	}
	if targetActiveAdmin && activeAdmins <= 1 {
		return identity.ErrLastAdmin
	}
	return s.RevokeRole(context.Background(), targetID, identity.RoleAdmin)
}

// DisableMFAMethodSafely mirrors the production Last Active Admin decision for
// the shared fakeStore so service tests cover the self-service MFA-removal
// bypass without weakening the production fail-closed persistence boundary.
func (s *fakeStore) DisableMFAMethodSafely(ctx context.Context, userID string) error {
	targetActiveAdmin := false
	activeAdmins := 0
	for _, u := range s.users {
		isAdmin := false
		for _, role := range s.userRoles[u.ID] {
			if role == identity.RoleAdmin {
				isAdmin = true
				break
			}
		}
		if isAdmin && u.Status == identity.StatusActive {
			activeAdmins++
			if u.ID == userID {
				targetActiveAdmin = true
			}
		}
	}
	if targetActiveAdmin && activeAdmins <= 1 {
		return identity.ErrLastAdmin
	}
	return s.DisableMFAMethod(ctx, userID)
}

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

func TestDisableTOTPAllowsAdminWhenAnotherActiveAdminExists(t *testing.T) {
	store := newFakeStore()
	first := identity.User{ID: "admin-1", Email: "admin1@example.com", Status: identity.StatusActive}
	second := identity.User{ID: "admin-2", Email: "admin2@example.com", Status: identity.StatusActive}
	store.users[first.Email] = first
	store.users[second.Email] = second
	store.userRoles[first.ID] = []string{identity.RoleAdmin}
	store.userRoles[second.ID] = []string{identity.RoleAdmin}
	store.mfaMethods[first.ID] = &identity.MFAMethod{Secret: "secret", Confirmed: true}

	svc := identity.NewAuthService(store)
	if err := svc.DisableTOTP(context.Background(), first.ID); err != nil {
		t.Fatalf("disable MFA with another active admin should succeed, got %v", err)
	}
	if !store.mfaMethods[first.ID].Disabled {
		t.Fatal("expected target MFA to be disabled")
	}
}
