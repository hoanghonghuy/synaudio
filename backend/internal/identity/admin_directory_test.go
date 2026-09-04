package identity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

type adminDirectoryFakeStore struct {
	*fakeStore
	usersResult []identity.AdminUserSummary
	lastQuery   string
	lastStatus  string
	lastLimit   int
}

func (s *adminDirectoryFakeStore) ListAdminUsers(_ context.Context, query, status string, limit int) ([]identity.AdminUserSummary, error) {
	s.lastQuery = query
	s.lastStatus = status
	s.lastLimit = limit
	return s.usersResult, nil
}

func (s *adminDirectoryFakeStore) GetAdminUser(_ context.Context, userID string) (identity.AdminUserSummary, error) {
	for _, user := range s.usersResult {
		if user.ID == userID {
			return user, nil
		}
	}
	return identity.AdminUserSummary{}, identity.ErrUserNotFound
}

func TestAdminUserDirectoryRequiresOperationPermission(t *testing.T) {
	store := &adminDirectoryFakeStore{fakeStore: newFakeStore()}
	store.userRoles["actor"] = []string{identity.RoleAdmin}
	store.rolePermissions[identity.RoleAdmin] = []string{identity.PermAdminRoleGrant}
	svc := identity.NewAuthService(store)

	_, err := svc.ListAdminUsers(context.Background(), "actor", "", "", 50)
	if !errors.Is(err, identity.ErrForbidden) {
		t.Fatalf("expected permission denial, got %v", err)
	}
}

func TestAdminUserDirectoryNormalizesFiltersAndBoundsLimit(t *testing.T) {
	store := &adminDirectoryFakeStore{fakeStore: newFakeStore()}
	store.userRoles["actor"] = []string{identity.RoleAdmin}
	store.rolePermissions[identity.RoleAdmin] = []string{identity.PermUserStatusManage}
	store.usersResult = []identity.AdminUserSummary{{ID: "target", Email: "target@example.com", Status: identity.StatusActive}}
	svc := identity.NewAuthService(store)

	users, err := svc.ListAdminUsers(context.Background(), "actor", "  target@example.com  ", " active ", 1000)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(users) != 1 || users[0].ID != "target" {
		t.Fatalf("unexpected users: %#v", users)
	}
	if store.lastQuery != "target@example.com" || store.lastStatus != identity.StatusActive || store.lastLimit != 100 {
		t.Fatalf("unexpected normalized args: query=%q status=%q limit=%d", store.lastQuery, store.lastStatus, store.lastLimit)
	}
}

func TestAdminUserDirectoryRejectsUnknownStatus(t *testing.T) {
	store := &adminDirectoryFakeStore{fakeStore: newFakeStore()}
	store.userRoles["actor"] = []string{identity.RoleAdmin}
	store.rolePermissions[identity.RoleAdmin] = []string{identity.PermUserStatusManage}
	svc := identity.NewAuthService(store)

	if _, err := svc.ListAdminUsers(context.Background(), "actor", "", "deleted", 50); err == nil || err.Error() != "invalid account status" {
		t.Fatalf("expected invalid status, got %v", err)
	}
}
