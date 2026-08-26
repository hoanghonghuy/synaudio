package identity

import (
	"context"
)

// RequestAccountDeletion deactivates the account immediately, starting the
// grace period before purge.
func (s *AuthService) RequestAccountDeletion(ctx context.Context, userID string) error {
	if _, err := s.store.GetUserByID(ctx, userID); err != nil {
		return err
	}
	return s.store.DeactivateUser(ctx, userID)
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
