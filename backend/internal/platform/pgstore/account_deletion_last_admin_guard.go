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
	var targetIsAdmin bool
	if err := tx.QueryRow(ctx, `
SELECT u.status,
       EXISTS (
           SELECT 1
             FROM user_roles ur
             JOIN roles r ON r.id = ur.role_id
            WHERE ur.user_id = u.id
              AND r.code = 'ADMIN'
       )
  FROM users u
 WHERE u.id = $1
 FOR UPDATE
`, toUUID(userID)).Scan(&currentStatus, &targetIsAdmin); err != nil {
		return identity.ErrUserNotFound
	}

	if currentStatus == identity.StatusActive && targetIsAdmin {
		var activeAdmins int
		if err := tx.QueryRow(ctx, `
SELECT COUNT(*)
  FROM user_roles ur
  JOIN roles r ON r.id = ur.role_id
  JOIN users u ON u.id = ur.user_id
 WHERE r.code = 'ADMIN'
   AND u.status = 'ACTIVE'
`).Scan(&activeAdmins); err != nil {
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
