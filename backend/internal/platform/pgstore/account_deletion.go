package pgstore

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

// RequestAccountDeletion deactivates the account and revokes every active
// session in one statement so no second device retains effective access after
// the deletion request commits.
func (s *IdentityStore) RequestAccountDeletion(ctx context.Context, userID string) error {
	var applied bool
	err := s.q.DBTX().QueryRow(ctx, `
WITH deactivated AS (
    UPDATE users
       SET status = 'DEACTIVATED',
           deactivated_at = COALESCE(deactivated_at, NOW()),
           updated_at = NOW()
     WHERE id = $1
       AND status <> 'DEACTIVATED'
    RETURNING id
), revoked AS (
    UPDATE user_sessions
       SET revoked_at = NOW()
     WHERE user_id = $1
       AND revoked_at IS NULL
    RETURNING id
)
SELECT EXISTS (SELECT 1 FROM deactivated)
`, toUUID(userID)).Scan(&applied)
	if err != nil {
		return err
	}
	if !applied {
		// Idempotent repeat requests for an already-deactivated existing user are
		// accepted; only a missing user is an error.
		var exists bool
		if err := s.q.DBTX().QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, toUUID(userID)).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return identity.ErrUserNotFound
		}
	}
	return nil
}

// PurgeAccountIfEligible performs the V1 security-material cleanup and PII
// anonymization only after the frozen 30-day grace cutoff. The data-modifying
// CTEs are one PostgreSQL statement, so a failure cannot leave a partial purge.
func (s *IdentityStore) PurgeAccountIfEligible(ctx context.Context, userID string, cutoff time.Time) error {
	var applied bool
	err := s.q.DBTX().QueryRow(ctx, `
WITH eligible AS (
    SELECT id
      FROM users
     WHERE id = $1
       AND status = 'DEACTIVATED'
       AND deactivated_at IS NOT NULL
       AND deactivated_at <= $2
     FOR UPDATE
), sessions_deleted AS (
    DELETE FROM user_sessions WHERE user_id = (SELECT id FROM eligible)
), mfa_deleted AS (
    DELETE FROM user_mfa_methods WHERE user_id = (SELECT id FROM eligible)
), verification_deleted AS (
    DELETE FROM email_verification_tokens WHERE user_id = (SELECT id FROM eligible)
), reset_deleted AS (
    DELETE FROM password_reset_tokens WHERE user_id = (SELECT id FROM eligible)
), roles_deleted AS (
    DELETE FROM user_roles WHERE user_id = (SELECT id FROM eligible)
), anonymized AS (
    UPDATE users
       SET email = 'deleted-' || id::text || '@deleted.invalid',
           password_hash = NULL,
           display_name = NULL,
           updated_at = NOW()
     WHERE id = (SELECT id FROM eligible)
    RETURNING id
)
SELECT EXISTS (SELECT 1 FROM anonymized)
`, toUUID(userID), cutoff).Scan(&applied)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	var status string
	var deactivatedAt *time.Time
	err = s.q.DBTX().QueryRow(ctx, `SELECT status, deactivated_at FROM users WHERE id = $1`, toUUID(userID)).Scan(&status, &deactivatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.ErrUserNotFound
	}
	if err != nil {
		return err
	}
	return identity.ErrDeletionGracePeriod
}

func (s *IdentityStore) ListPurgeEligibleAccounts(ctx context.Context, cutoff time.Time, limit int) ([]string, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.q.DBTX().Query(ctx, `
SELECT id::text
  FROM users
 WHERE status = 'DEACTIVATED'
   AND deactivated_at IS NOT NULL
   AND deactivated_at <= $1
 ORDER BY deactivated_at, id
 LIMIT $2
`, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
