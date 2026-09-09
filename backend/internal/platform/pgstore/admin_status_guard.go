package pgstore

import (
	"context"
	"errors"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

// SetUserStatusSafely serializes privileged status transitions with the Last
// Active Admin invariant. Moving an ACTIVE admin out of ACTIVE and the guard
// decision happen in one transaction; sessions are revoked in the same commit
// so suspended/deactivated privilege cannot linger through an old session.
func (s *IdentityStore) SetUserStatusSafely(ctx context.Context, targetID, status string) error {
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
`, toUUID(targetID)).Scan(&currentStatus, &targetIsAdmin); err != nil {
		return identity.ErrUserNotFound
	}

	if currentStatus == identity.StatusActive && status != identity.StatusActive && targetIsAdmin {
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

	tag, err := tx.Exec(ctx, `UPDATE users SET status = $2, updated_at = NOW() WHERE id = $1`, toUUID(targetID), status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return identity.ErrUserNotFound
	}

	if status != identity.StatusActive {
		if _, err := tx.Exec(ctx, `UPDATE user_sessions SET revoked_at = COALESCE(revoked_at, NOW()) WHERE user_id = $1`, toUUID(targetID)); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
