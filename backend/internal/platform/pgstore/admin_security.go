package pgstore

import (
	"context"
	"errors"
	"time"

	"github.com/synaudio/synaudio/backend/internal/identity"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

func (s *IdentityStore) ReplaceMFARecoveryCodes(ctx context.Context, userID string, codeHashes []string) error {
	return replaceMFARecoveryCodes(ctx, s.q.DBTX(), userID, codeHashes)
}

func (s *IdentityStore) ConfirmMFAWithRecoveryCodes(ctx context.Context, userID string, codeHashes []string) error {
	beginner, ok := s.q.DBTX().(transactionBeginner)
	if !ok {
		return errors.New("identity store transaction support unavailable")
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := replaceMFARecoveryCodes(ctx, tx, userID, codeHashes); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE user_mfa_methods SET confirmed_at = NOW() WHERE user_id = $1 AND confirmed_at IS NULL`, toUUID(userID)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *IdentityStore) ConfirmMFAWithRecoveryCodesAndSession(ctx context.Context, userID, sessionID string, codeHashes []string, at time.Time) error {
	beginner, ok := s.q.DBTX().(transactionBeginner)
	if !ok {
		return errors.New("identity store transaction support unavailable")
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := replaceMFARecoveryCodes(ctx, tx, userID, codeHashes); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE user_mfa_methods SET confirmed_at = NOW() WHERE user_id = $1 AND confirmed_at IS NULL`, toUUID(userID)); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
UPDATE user_sessions
   SET mfa_verified_at = $3,
       recent_auth_at = $3
 WHERE id = $1
   AND user_id = $2
   AND revoked_at IS NULL
   AND expires_at > $3
`, toUUID(sessionID), toUUID(userID), at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return identity.ErrUnauthenticated
	}
	return tx.Commit(ctx)
}

func replaceMFARecoveryCodes(ctx context.Context, executor db.DBTX, userID string, codeHashes []string) error {
	args := make([]interface{}, 0, len(codeHashes)+1)
	args = append(args, toUUID(userID))
	values := ""
	for i, hash := range codeHashes {
		if i > 0 {
			values += ","
		}
		values += "($1,$" + decimal(i+2) + ")"
		args = append(args, hash)
	}
	if values == "" {
		_, err := executor.Exec(ctx, `DELETE FROM user_mfa_recovery_codes WHERE user_id = $1`, toUUID(userID))
		return err
	}
	_, err := executor.Exec(ctx, `
WITH removed AS (
    DELETE FROM user_mfa_recovery_codes WHERE user_id = $1
)
INSERT INTO user_mfa_recovery_codes (id, user_id, code_hash)
SELECT gen_random_uuid(), v.user_id::uuid, v.code_hash
FROM (VALUES `+values+`) AS v(user_id, code_hash)
`, args...)
	return err
}

func (s *IdentityStore) ConsumeMFARecoveryCode(ctx context.Context, userID, codeHash string) (bool, error) {
	var consumed bool
	err := s.q.DBTX().QueryRow(ctx, `
WITH consumed AS (
    UPDATE user_mfa_recovery_codes
       SET used_at = NOW()
     WHERE id = (
         SELECT id
           FROM user_mfa_recovery_codes
          WHERE user_id = $1
            AND code_hash = $2
            AND used_at IS NULL
          ORDER BY id
          LIMIT 1
          FOR UPDATE
     )
    RETURNING id
)
SELECT EXISTS (SELECT 1 FROM consumed)
`, toUUID(userID), codeHash).Scan(&consumed)
	return consumed, err
}

func (s *IdentityStore) MarkSessionMFAAndRecentAuth(ctx context.Context, userID, sessionID string, at time.Time) error {
	tag, err := s.q.DBTX().Exec(ctx, `
UPDATE user_sessions
   SET mfa_verified_at = $3,
       recent_auth_at = $3
 WHERE id = $1
   AND user_id = $2
   AND revoked_at IS NULL
   AND expires_at > $3
`, toUUID(sessionID), toUUID(userID), at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return identity.ErrUnauthenticated
	}
	return nil
}

func (s *IdentityStore) HasPrivilegedSessionAssurance(ctx context.Context, userID, sessionID string, now time.Time) (bool, error) {
	var allowed bool
	err := s.q.DBTX().QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
      FROM users u
      JOIN user_sessions s ON s.user_id = u.id
      JOIN user_mfa_methods m ON m.user_id = u.id
     WHERE u.id = $1
       AND s.id = $2
       AND u.status = 'ACTIVE'
       AND u.email_verified_at IS NOT NULL
       AND s.revoked_at IS NULL
       AND s.expires_at > $3
       AND s.mfa_verified_at IS NOT NULL
       AND m.confirmed_at IS NOT NULL
       AND m.disabled_at IS NULL
)
`, toUUID(userID), toUUID(sessionID), now).Scan(&allowed)
	return allowed, err
}

func (s *IdentityStore) HasRecentAuth(ctx context.Context, userID, sessionID string, cutoff time.Time) (bool, error) {
	var allowed bool
	err := s.q.DBTX().QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM user_sessions
     WHERE id = $1
       AND user_id = $2
       AND revoked_at IS NULL
       AND recent_auth_at IS NOT NULL
       AND recent_auth_at >= $3
)
`, toUUID(sessionID), toUUID(userID), cutoff).Scan(&allowed)
	return allowed, err
}

func decimal(value int) string {
	if value == 0 {
		return "0"
	}
	buf := [20]byte{}
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[i:])
}
