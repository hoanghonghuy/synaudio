package identity

import (
	"context"
	"errors"
	"net/http"
	"time"
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

func (s *AuthService) privilegedPrincipal(ctx context.Context, r *http.Request) (Principal, User, bool, error) {
	principal, user, err := s.AuthenticateRequest(ctx, r)
	if err != nil {
		return Principal{}, User{}, false, err
	}
	if user.Status != StatusActive || user.EmailVerifiedAt == "" {
		return principal, user, false, nil
	}
	roles, err := s.store.GetUserRoles(ctx, principal.UserID)
	if err != nil {
		return principal, user, false, err
	}
	admin := false
	for _, role := range roles {
		if role == RoleAdmin {
			admin = true
			break
		}
	}
	if !admin {
		return principal, user, false, nil
	}
	securityStore, ok := s.store.(mfaSecurityStore)
	if !ok {
		return principal, user, false, errors.New("privileged security persistence not configured")
	}
	assured, err := securityStore.HasPrivilegedSessionAssurance(ctx, principal.UserID, principal.SessionID, s.settings.Now().UTC())
	return principal, user, assured, err
}

// ResolveAdmin is the broad privileged boundary. ADMIN role alone is never
// sufficient: the account must remain ACTIVE and verified, MFA must currently
// be enabled, and this exact logical session must carry MFA assurance.
func (s *AuthService) ResolveAdmin(ctx context.Context, r *http.Request) (bool, error) {
	_, _, allowed, err := s.privilegedPrincipal(ctx, r)
	return allowed, err
}

// ResolveAdminPermission combines privileged-session assurance with a concrete
// operation permission. Routers should prefer this over a role-only admin gate.
func (s *AuthService) ResolveAdminPermission(ctx context.Context, r *http.Request, permission string) (bool, error) {
	principal, _, allowed, err := s.privilegedPrincipal(ctx, r)
	if err != nil || !allowed {
		return false, err
	}
	return s.Authorize(ctx, principal.UserID, permission)
}

// RequireRecentAuth enforces the frozen 10-minute high-risk action window.
// Successful TOTP confirmation refreshes recent_auth_at for the exact session.
func (s *AuthService) RequireRecentAuth(ctx context.Context, r *http.Request) error {
	principal, _, allowed, err := s.privilegedPrincipal(ctx, r)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	securityStore, ok := s.store.(mfaSecurityStore)
	if !ok {
		return ErrForbidden
	}
	fresh, err := securityStore.HasRecentAuth(ctx, principal.UserID, principal.SessionID, s.settings.Now().UTC().Add(-10*time.Minute))
	if err != nil {
		return err
	}
	if !fresh {
		return ErrForbidden
	}
	return nil
}

// GrantAdmin grants the ADMIN role to the target user, requiring the actor to
// already hold the operation-specific permission.
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
