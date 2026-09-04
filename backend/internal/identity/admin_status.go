package identity

import (
	"context"
	"errors"
)

const PermAdminStatusManage = "ADMIN_STATUS_MANAGE"

// AdminStatusGuardStore owns privileged account-status transitions that can
// affect the Last Active Admin invariant. Implementations must serialize the
// last-admin decision with the status write so concurrent suspensions or
// deactivations cannot remove every active administrator.
type AdminStatusGuardStore interface {
	SetUserStatusSafely(ctx context.Context, targetID, status string) error
}

// SetUserStatusAsAdmin changes a user's account status through the privileged
// security boundary. The caller must already be authorized for the concrete
// ADMIN_STATUS_MANAGE operation by the HTTP boundary; this service-level check
// is defense in depth for non-HTTP callers.
func (s *AuthService) SetUserStatusAsAdmin(ctx context.Context, actorID, targetID, status string) error {
	if ok, err := s.Authorize(ctx, actorID, PermAdminStatusManage); err != nil {
		return err
	} else if !ok {
		return ErrForbidden
	}

	switch status {
	case StatusActive, StatusSuspended, StatusDeactivated:
	default:
		return errors.New("invalid account status")
	}

	guardStore, ok := s.store.(AdminStatusGuardStore)
	if !ok {
		return errors.New("atomic admin status guard persistence not configured")
	}
	return guardStore.SetUserStatusSafely(ctx, targetID, status)
}
