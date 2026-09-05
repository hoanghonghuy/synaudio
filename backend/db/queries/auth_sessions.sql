-- ============================================================
-- V1 refresh session lifecycle
-- ============================================================

-- name: CreateAuthSession :one
INSERT INTO user_sessions (
    id, user_id, refresh_token_hash, created_at, last_used_at, expires_at,
    recent_auth_at, user_agent_summary, safe_ip_metadata
)
VALUES ($1, $2, $3, $4, $4, $5, $4, $6, $7)
RETURNING id, user_id, refresh_token_hash, created_at, last_used_at, expires_at,
          revoked_at, mfa_verified_at, recent_auth_at, user_agent_summary, safe_ip_metadata;

-- name: RotateAuthSession :one
UPDATE user_sessions
SET refresh_token_hash = sqlc.arg('new_refresh_hash'),
    last_used_at = sqlc.arg('now')
WHERE refresh_token_hash = sqlc.arg('refresh_token_hash')
  AND revoked_at IS NULL
  AND expires_at > sqlc.arg('now')
  AND COALESCE(last_used_at, created_at) >= sqlc.arg('idle_cutoff')
RETURNING id, user_id, refresh_token_hash, created_at, last_used_at, expires_at,
          revoked_at, mfa_verified_at, recent_auth_at, user_agent_summary, safe_ip_metadata;

-- name: TouchAuthSession :one
UPDATE user_sessions
SET last_used_at = sqlc.arg('now')
WHERE id = sqlc.arg('id')
  AND revoked_at IS NULL
  AND expires_at > sqlc.arg('now')
  AND COALESCE(last_used_at, created_at) >= sqlc.arg('idle_cutoff')
RETURNING id, user_id, refresh_token_hash, created_at, last_used_at, expires_at,
          revoked_at, mfa_verified_at, recent_auth_at, user_agent_summary, safe_ip_metadata;

-- name: RevokeAuthSessionByHash :execrows
UPDATE user_sessions
SET revoked_at = NOW()
WHERE refresh_token_hash = $1
  AND revoked_at IS NULL;

-- name: RevokeAuthSessionByIDForUser :execrows
UPDATE user_sessions
SET revoked_at = NOW()
WHERE id = $1
  AND user_id = $2
  AND revoked_at IS NULL;

-- name: ListActiveAuthSessions :many
SELECT id, user_id, refresh_token_hash, created_at, last_used_at, expires_at,
       revoked_at, mfa_verified_at, recent_auth_at, user_agent_summary, safe_ip_metadata
FROM user_sessions
WHERE user_id = sqlc.arg('user_id')
  AND revoked_at IS NULL
  AND expires_at > sqlc.arg('now')
  AND COALESCE(last_used_at, created_at) >= sqlc.arg('idle_cutoff')
ORDER BY last_used_at DESC NULLS LAST, created_at DESC;