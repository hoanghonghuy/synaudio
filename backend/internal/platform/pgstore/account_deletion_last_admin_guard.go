package pgstore

import (
	"context"
	"errors"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

// RequestAccountDeletionSafely serializes deletion deactivation with the Last
// Active Admin invariant and revokes sessions in the same transaction.
func (s *IdentityStore) RequestAccountDeletionSafely(ctx context.Context, userID string) error {
	beginner, ok := s.q.DBTX().(transactionBeginner)
	if !ok {
		return errors.New("identity store transaction support unavailable")
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('synaudio:last-active-admin'))`); err != nil {
		return err
	}

	var currentStatus string
	if err := tx.QueryRow(ctx, `
SELECT u.status
  FROM users u
 WHERE u.id = $1
 FOR UPDATE
`, toUUID(userID)).Scan(&currentStatus); err != nil {
		return identity.ErrUserNotFound
	}

	targetIsMfaCapable, err := targetIsMfaCapableActiveAdmin(ctx, tx, userID)
	if err != nil {
		return err
	}
	if currentStatus == identity.StatusActive && targetIsMfaCapable {
		activeAdmins, err := countMfaCapableActiveAdmins(ctx, tx)
		if err != nil {
			return err
		}
		if activeAdmins <= 1 {
			return identity.ErrLastAdmin
		}
	}

	tag, err := tx.Exec(ctx, `
UPDATE users
   SET status = 'DEACTIVATED',
       deactivated_at = COALESCE(deactivated_at, NOW()),
       updated_at = NOW()
 WHERE id = $1
`, toUUID(userID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return identity.ErrUserNotFound
	}

	if _, err := tx.Exec(ctx, `UPDATE user_sessions SET revoked_at = COALESCE(revoked_at, NOW()) WHERE user_id = $1`, toUUID(userID)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
