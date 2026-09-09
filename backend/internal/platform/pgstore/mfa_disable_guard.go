package pgstore

import (
	"context"
	"errors"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

// DisableMFAMethodSafely serializes MFA removal with the Last Active Admin
// invariant. The final ACTIVE ADMIN may not lose mandatory MFA through the
// ordinary self-service disable path.
func (s *IdentityStore) DisableMFAMethodSafely(ctx context.Context, userID string) error {
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

	var status string
	var isAdmin bool
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
`, toUUID(userID)).Scan(&status, &isAdmin); err != nil {
		return identity.ErrUserNotFound
	}

	if status == identity.StatusActive && isAdmin {
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

	if _, err := tx.Exec(ctx, `
UPDATE user_mfa_methods
   SET disabled_at = NOW()
 WHERE user_id = $1
   AND disabled_at IS NULL
`, toUUID(userID)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
