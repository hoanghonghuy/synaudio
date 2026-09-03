package pgstore

import (
	"context"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

// ReplaceVerificationToken invalidates every older outstanding verification
// credential before inserting the new one. The single statement keeps resend
// supersession atomic even outside a larger request transaction.
func (s *IdentityStore) ReplaceVerificationToken(ctx context.Context, userID, tokenHash string) error {
	_, err := s.q.DBTX().Exec(ctx, `
WITH invalidated AS (
    UPDATE email_verification_tokens
       SET used_at = NOW()
     WHERE user_id = $1
       AND used_at IS NULL
)
INSERT INTO email_verification_tokens (id, user_id, token_hash, expires_at)
VALUES (gen_random_uuid(), $1, $2, NOW() + INTERVAL '24 hours')
`, toUUID(userID), tokenHash)
	return err
}

// ReplaceResetToken gives password recovery the same one-active-credential
// invariant: requesting a new reset credential permanently supersedes older links.
func (s *IdentityStore) ReplaceResetToken(ctx context.Context, userID, tokenHash string) error {
	_, err := s.q.DBTX().Exec(ctx, `
WITH invalidated AS (
    UPDATE password_reset_tokens
       SET used_at = NOW()
     WHERE user_id = $1
       AND used_at IS NULL
)
INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at)
VALUES (gen_random_uuid(), $1, $2, NOW() + INTERVAL '1 hour')
`, toUUID(userID), tokenHash)
	return err
}

// ConsumeVerificationToken atomically consumes the exact latest credential and
// marks the account verified. A concurrent/replayed request observes no matching
// unconsumed row and therefore cannot succeed twice.
func (s *IdentityStore) ConsumeVerificationToken(ctx context.Context, userID, tokenHash string) error {
	var applied bool
	err := s.q.DBTX().QueryRow(ctx, `
WITH consumed AS (
    UPDATE email_verification_tokens
       SET used_at = NOW()
     WHERE id = (
        SELECT id
          FROM email_verification_tokens
         WHERE user_id = $1
           AND token_hash = $2
           AND used_at IS NULL
           AND expires_at > NOW()
         ORDER BY created_at DESC
         LIMIT 1
     )
    RETURNING user_id
), verified AS (
    UPDATE users
       SET email_verified_at = COALESCE(email_verified_at, NOW()),
           updated_at = NOW()
     WHERE id = (SELECT user_id FROM consumed)
    RETURNING id
)
SELECT EXISTS (SELECT 1 FROM verified)
`, toUUID(userID), tokenHash).Scan(&applied)
	if err != nil {
		return err
	}
	if !applied {
		return identity.ErrInvalidToken
	}
	return nil
}

// ResetPasswordAtomic consumes the reset credential, changes the password and
// revokes active sessions in one PostgreSQL statement. Any statement failure
// rolls the whole recovery transition back.
func (s *IdentityStore) ResetPasswordAtomic(ctx context.Context, userID, tokenHash, passwordHash string) error {
	var applied bool
	err := s.q.DBTX().QueryRow(ctx, `
WITH consumed AS (
    UPDATE password_reset_tokens
       SET used_at = NOW()
     WHERE id = (
        SELECT id
          FROM password_reset_tokens
         WHERE user_id = $1
           AND token_hash = $2
           AND used_at IS NULL
           AND expires_at > NOW()
         ORDER BY created_at DESC
         LIMIT 1
     )
    RETURNING user_id
), password_updated AS (
    UPDATE users
       SET password_hash = $3,
           updated_at = NOW()
     WHERE id = (SELECT user_id FROM consumed)
    RETURNING id
), sessions_revoked AS (
    UPDATE user_sessions
       SET revoked_at = NOW()
     WHERE user_id = (SELECT id FROM password_updated)
       AND revoked_at IS NULL
    RETURNING id
)
SELECT EXISTS (SELECT 1 FROM password_updated)
`, toUUID(userID), tokenHash, passwordHash).Scan(&applied)
	if err != nil {
		return err
	}
	if !applied {
		return identity.ErrInvalidToken
	}
	return nil
}
