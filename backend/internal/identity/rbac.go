package identity

import (
	"context"
)

// Authorize reports whether the user holds the given permission through any
// of their roles.
func (s *AuthService) Authorize(ctx context.Context, userID, permission string) (bool, error) {
	roles, err := s.store.GetUserRoles(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, role := range roles {
		perms, err := s.store.GetRolePermissions(ctx, role)
		if err != nil {
			return false, err
		}
		for _, p := range perms {
			if p == permission {
				return true, nil
			}
		}
	}

	return false, nil
}

// GrantAdmin grants the ADMIN role to the target user, requiring the actor to
// already hold the ADMIN role.
func (s *AuthService) GrantAdmin(ctx context.Context, actorID, targetID string) error {
	if ok, err := s.Authorize(ctx, actorID, PermAdminRoleGrant); err != nil {
		return err
	} else if !ok {
		return ErrForbidden
	}

	return s.store.GrantRole(ctx, targetID, RoleAdmin)
}

// RevokeAdmin removes the ADMIN role from the target user, enforcing the
// Last Active Admin Guard.
func (s *AuthService) RevokeAdmin(ctx context.Context, actorID, targetID string) error {
	if ok, err := s.Authorize(ctx, actorID, PermAdminRoleRevoke); err != nil {
		return err
	} else if !ok {
		return ErrForbidden
	}

	count, err := s.store.CountActiveAdmins(ctx)
	if err != nil {
		return err
	}
	if count <= 1 {
		return ErrLastAdmin
	}

	return s.store.RevokeRole(ctx, targetID, RoleAdmin)
}
