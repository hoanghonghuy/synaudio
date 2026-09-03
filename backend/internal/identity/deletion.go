package identity

import (
	"context"
	"errors"
	"time"
)

const (
	AccountDeletionGracePeriod = 30 * 24 * time.Hour
	AccountDeletionRecoveryTTL = 30 * time.Minute
)

var ErrDeletionGracePeriod = errors.New("account deletion grace period has not elapsed")

type deletionLifecycleStore interface {
	RequestAccountDeletion(ctx context.Context, userID string) error
	PurgeAccountIfEligible(ctx context.Context, userID string, cutoff time.Time) error
	ListPurgeEligibleAccounts(ctx context.Context, cutoff time.Time, limit int) ([]string, error)
}

type deletionRecoveryStore interface {
	ReplaceDeletionRecoveryToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	CancelAccountDeletionWithToken(ctx context.Context, userID, tokenHash string, now time.Time) error
}

type AccountDeletionPurgeEvent struct {
	UserID  string
	Outcome string
	Err     error
}

type AccountDeletionPurgeObserver func(context.Context, AccountDeletionPurgeEvent) error

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

// RequestAccountDeletionRecovery issues a short-lived, one-time ownership proof
// only for an account that is still inside the deletion grace period. Callers
// must keep the externally observable response enumeration-safe.
func (s *AuthService) RequestAccountDeletionRecovery(ctx context.Context, email string) (string, error) {
	u, err := s.store.GetUserByEmail(ctx, NormalizeEmail(email))
	if err != nil || u.Status != "DEACTIVATED" {
		return "", ErrUserNotFound
	}
	recoveryStore, ok := s.store.(deletionRecoveryStore)
	if !ok {
		return "", errors.New("account deletion recovery persistence not configured")
	}
	raw, err := NewRefreshToken()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	if err := recoveryStore.ReplaceDeletionRecoveryToken(ctx, u.ID, HashToken(raw), now.Add(AccountDeletionRecoveryTTL)); err != nil {
		return "", err
	}
	return raw, nil
}

// CancelAccountDeletionByEmail proves ownership with the dedicated one-time
// deletion-recovery token and atomically consumes it while reactivating the
// account. No ordinary authenticated capability is granted to DEACTIVATED users.
func (s *AuthService) CancelAccountDeletionByEmail(ctx context.Context, email, token string) error {
	u, err := s.store.GetUserByEmail(ctx, NormalizeEmail(email))
	if err != nil || u.Status != "DEACTIVATED" {
		return ErrInvalidToken
	}
	recoveryStore, ok := s.store.(deletionRecoveryStore)
	if !ok {
		return ErrInvalidToken
	}
	return recoveryStore.CancelAccountDeletionWithToken(ctx, u.ID, HashToken(token), time.Now().UTC())
}

// CancelAccountDeletion remains an internal service operation for privileged or
// already-authorized coordination paths. Public recovery for a DEACTIVATED user
// must use CancelAccountDeletionByEmail instead of weakening normal Bearer auth.
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

// PurgeEligibleAccounts preserves the original reconciliation API for callers
// that do not need per-account delivery evidence.
func (s *AuthService) PurgeEligibleAccounts(ctx context.Context, limit int) (int, error) {
	return s.PurgeEligibleAccountsObserved(ctx, limit, nil)
}

// PurgeEligibleAccountsObserved is the operational reconciliation boundary. It
// reports each attempted purge without exposing PII or recovery credentials so
// the worker can append durable lifecycle audit evidence.
func (s *AuthService) PurgeEligibleAccountsObserved(ctx context.Context, limit int, observe AccountDeletionPurgeObserver) (int, error) {
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
		err := lifecycle.PurgeAccountIfEligible(ctx, id, cutoff)
		if err != nil {
			if observe != nil {
				_ = observe(ctx, AccountDeletionPurgeEvent{UserID: id, Outcome: "FAILED", Err: err})
			}
			if errors.Is(err, ErrDeletionGracePeriod) || errors.Is(err, ErrUserNotFound) {
				continue
			}
			return purged, err
		}
		purged++
		if observe != nil {
			if err := observe(ctx, AccountDeletionPurgeEvent{UserID: id, Outcome: "SUCCEEDED"}); err != nil {
				return purged, err
			}
		}
	}
	return purged, nil
}
