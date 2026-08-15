package identity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func TestAuthorizeReturnsTrueWhenRoleHasPermission(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, _ := svc.Register(context.Background(), "admin@example.com", "correct password")
	store.userRoles[u.ID] = []string{identity.RoleAdmin}
	store.rolePermissions[identity.RoleAdmin] = []string{identity.PermStoryCreate}

	ok, err := svc.Authorize(context.Background(), u.ID, identity.PermStoryCreate)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if !ok {
		t.Fatal("expected permission granted")
	}
}

func TestAuthorizeReturnsFalseWhenPermissionMissing(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	u, _ := svc.Register(context.Background(), "user@example.com", "correct password")
	store.userRoles[u.ID] = []string{identity.RoleUser}
	store.rolePermissions[identity.RoleUser] = []string{}

	ok, err := svc.Authorize(context.Background(), u.ID, identity.PermStoryCreate)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if ok {
		t.Fatal("expected permission denied")
	}
}

func TestGrantAdminAddsRole(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	actor, _ := svc.Register(context.Background(), "admin@example.com", "correct password")
	target, _ := svc.Register(context.Background(), "user@example.com", "correct password")
	store.userRoles[actor.ID] = []string{identity.RoleAdmin}
	store.rolePermissions[identity.RoleAdmin] = []string{identity.PermAdminRoleGrant}

	if err := svc.GrantAdmin(context.Background(), actor.ID, target.ID); err != nil {
		t.Fatalf("grant admin: %v", err)
	}

	roles := store.userRoles[target.ID]
	if !contains(roles, identity.RoleAdmin) {
		t.Fatalf("expected ADMIN role, got %v", roles)
	}
}

func TestRevokeAdminRemovesRole(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	actor, _ := svc.Register(context.Background(), "admin1@example.com", "correct password")
	target, _ := svc.Register(context.Background(), "admin2@example.com", "correct password")
	store.userRoles[actor.ID] = []string{identity.RoleAdmin}
	store.userRoles[target.ID] = []string{identity.RoleAdmin}
	store.rolePermissions[identity.RoleAdmin] = []string{identity.PermAdminRoleRevoke}

	if err := svc.RevokeAdmin(context.Background(), actor.ID, target.ID); err != nil {
		t.Fatalf("revoke admin: %v", err)
	}

	if contains(store.userRoles[target.ID], identity.RoleAdmin) {
		t.Fatal("expected ADMIN role removed")
	}
}

func TestRevokeAdminFailsWhenLastAdmin(t *testing.T) {
	store := newFakeStore()
	svc := identity.NewAuthService(store)

	actor, _ := svc.Register(context.Background(), "admin@example.com", "correct password")
	store.userRoles[actor.ID] = []string{identity.RoleAdmin}
	store.rolePermissions[identity.RoleAdmin] = []string{identity.PermAdminRoleRevoke}

	err := svc.RevokeAdmin(context.Background(), actor.ID, actor.ID)
	if !errors.Is(err, identity.ErrLastAdmin) {
		t.Fatalf("expected ErrLastAdmin, got %v", err)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
