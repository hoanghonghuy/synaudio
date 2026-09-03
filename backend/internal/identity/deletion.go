package identity

import (
	"context"
	"errors"
	"time"
)

const AccountDeletionGracePeriod = 30 * 24 * time.Hour

var ErrDeletionGracePeriod = errors.New("account deletion grace period has not elapsed")

type deletionLifecycleStore interface {
	RequestAccountDeletion(ctx context.Context, userID string) error
	PurgeAccountIfEligible(ctx context.Context, userID string, cutoff time.Time) error
	ListPurgeEligibleAccounts(ctx context.Context, cutoff time.Time, limit int) ([]string, error)
}

// RequestAccountDeletion deactivates the account immediately and revokes active
// sessions. The production store performs both transitions atomically.
func (s *AuthService) RequestAccountDeletion(ctx context.Context, userID string) error {
	if _, err := s.store.GetUserByID(ctx, userID); err != nil {
		return err
	}
	if lifecycle, ok := s.store.(deletionLifecycleStore); ok {
		return lifecycle.RequestAccountDeletion(ctx, userID)
	}
	if err := s.store.DeactivateUser(ctx, userID); err != nil {
		return err
	}
	return s.store.RevokeSessions(ctx, userID)
}

// CancelAccountDeletion remains an internal service operation for now. The
// public ACTIVE-session route cannot safely recover a DEACTIVATED user; #20's
// dedicated ownership-proof recovery path is wired separately once the shared
// transactional-email capability is available on develop.
func (s *AuthService) CancelAccountDeletion(ctx context.Context, userID string) error {
	if _, err := s.store.GetUserByID(ctx, userID); err != nil {
		return err
	}
	return s.store.ReactivateUser(ctx, userID)
}

// PurgeAccount enforces the frozen 30-day grace period in the production store
// before anonymizing PII and removing active security material.
func (s *AuthService) PurgeAccount(ctx context.Context, userID string) error {
	if _, err := s.store.GetUserByID(ctx, userID); err != nil {
		return err
	}
	if lifecycle, ok := s.store.(deletionLifecycleStore); ok {
		return lifecycle.PurgeAccountIfEligible(ctx, userID, time.Now().UTC().Add(-AccountDeletionGracePeriod))
	}
	return s.store.PurgeUser(ctx, userID)
}

// PurgeEligibleAccounts is the worker reconciliation boundary. Each account is
// re-checked by PurgeAccountIfEligible so concurrent recovery cannot race an
// eligibility snapshot into an early purge.
func (s *AuthService) PurgeEligibleAccounts(ctx context.Context, limit int) (int, error) {
	lifecycle, ok := s.store.(deletionLifecycleStore)
	if !ok {
		return 0, errors.New("account deletion lifecycle persistence not configured")
	}
	cutoff := time.Now().UTC().Add(-AccountDeletionGracePeriod)
	ids, err := lifecycle.ListPurgeEligibleAccounts(ctx, cutoff, limit)
	if err != nil {
		return 0, err
	}
	purged := 0
	for _, id := range ids {
		if err := lifecycle.PurgeAccountIfEligible(ctx, id, cutoff); err != nil {
			if errors.Is(err, ErrDeletionGracePeriod) || errors.Is(err, ErrUserNotFound) {
				continue
			}
			return purged, err
		}
		purged++
	}
	return purged, nil
}
