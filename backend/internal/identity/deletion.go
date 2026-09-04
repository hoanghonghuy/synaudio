package identity

import (
	"context"
	"errors"
)

// accountDeletionGuardStore owns the transition that starts account deletion
// for accounts that may carry ADMIN. Implementations must serialize the Last
// Active Admin decision with deactivation so concurrent privileged transitions
// cannot remove every ACTIVE administrator.
type accountDeletionGuardStore interface {
	RequestAccountDeletionSafely(ctx context.Context, userID string) error
}

// RequestAccountDeletion deactivates the account immediately, starting the
// grace period before purge. The transition fails closed unless persistence can
// enforce the Last Active Admin invariant atomically with deactivation.
func (s *AuthService) RequestAccountDeletion(ctx context.Context, userID string) error {
	if _, err := s.store.GetUserByID(ctx, userID); err != nil {
		return err
	}
	guardStore, ok := s.store.(accountDeletionGuardStore)
	if !ok {
		return errors.New("atomic account deletion guard persistence not configured")
	}
	return guardStore.RequestAccountDeletionSafely(ctx, userID)
}

// CancelAccountDeletion restores a deactivated account to ACTIVE.
func (s *AuthService) CancelAccountDeletion(ctx context.Context, userID string) error {
	if _, err := s.store.GetUserByID(ctx, userID); err != nil {
		return err
	}
	return s.store.ReactivateUser(ctx, userID)
}

// PurgeAccount anonymizes PII after the grace period has elapsed.
func (s *AuthService) PurgeAccount(ctx context.Context, userID string) error {
	if _, err := s.store.GetUserByID(ctx, userID); err != nil {
		return err
	}
	return s.store.PurgeUser(ctx, userID)
}
