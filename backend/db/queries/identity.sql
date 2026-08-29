-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, display_name, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, email, password_hash, display_name, status, email_verified_at, created_at, updated_at, deactivated_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, display_name, status, email_verified_at, created_at, updated_at, deactivated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, display_name, status, email_verified_at, created_at, updated_at, deactivated_at
FROM users
WHERE id = $1;

-- name: CreateSession :exec
INSERT INTO user_sessions (id, user_id, refresh_token_hash, expires_at)
VALUES ($1, $2, $3, $4);

-- name: GetSessionByRefreshTokenHash :one
SELECT id, user_id, refresh_token_hash, expires_at
FROM user_sessions
WHERE refresh_token_hash = $1
  AND revoked_at IS NULL
  AND expires_at > NOW();

-- name: RevokeSession :exec
UPDATE user_sessions
SET revoked_at = NOW()
WHERE refresh_token_hash = $1
  AND revoked_at IS NULL;

-- name: StoreVerificationToken :exec
INSERT INTO email_verification_tokens (id, user_id, token_hash, expires_at)
VALUES (gen_random_uuid(), $1, $2, NOW() + INTERVAL '24 hours');

-- name: GetVerificationToken :one
SELECT token_hash
FROM email_verification_tokens
WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1;

-- name: MarkEmailVerified :exec
UPDATE users SET email_verified_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: StoreResetToken :exec
INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at)
VALUES (gen_random_uuid(), $1, $2, NOW() + INTERVAL '1 hour');

-- name: GetResetToken :one
SELECT token_hash
FROM password_reset_tokens
WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1;

-- name: UpdatePassword :exec
UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1;

-- name: RevokeSessions :exec
UPDATE user_sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL;

-- name: StoreMFAMethod :exec
INSERT INTO user_mfa_methods (id, user_id, type, encrypted_secret)
VALUES (gen_random_uuid(), $1, 'TOTP', $2);

-- name: GetMFAMethod :one
SELECT id, user_id, type, encrypted_secret, confirmed_at, disabled_at
FROM user_mfa_methods
WHERE user_id = $1
LIMIT 1;

-- name: ConfirmMFAMethod :exec
UPDATE user_mfa_methods SET confirmed_at = NOW() WHERE user_id = $1 AND confirmed_at IS NULL;

-- name: DisableMFAMethod :exec
UPDATE user_mfa_methods SET disabled_at = NOW() WHERE user_id = $1 AND disabled_at IS NULL;

-- name: GetUserRoles :many
SELECT r.code
FROM user_roles ur
JOIN roles r ON r.id = ur.role_id
WHERE ur.user_id = $1;

-- name: GetRolePermissions :many
SELECT p.code
FROM role_permissions rp
JOIN permissions p ON p.id = rp.permission_id
JOIN roles r ON r.id = rp.role_id
WHERE r.code = $1;

-- name: GrantRole :exec
INSERT INTO user_roles (user_id, role_id)
SELECT $1, id FROM roles WHERE code = $2
ON CONFLICT (user_id, role_id) DO NOTHING;

-- name: RevokeRole :exec
DELETE FROM user_roles
WHERE user_id = $1 AND role_id = (SELECT id FROM roles WHERE code = $2);

-- name: CountActiveAdmins :one
SELECT COUNT(*)
FROM user_roles ur
JOIN roles r ON r.id = ur.role_id
JOIN users u ON u.id = ur.user_id
WHERE r.code = 'ADMIN' AND u.status = 'ACTIVE';

-- name: DeactivateUser :exec
UPDATE users SET status = 'DEACTIVATED', deactivated_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: ReactivateUser :exec
UPDATE users SET status = 'ACTIVE', deactivated_at = NULL, updated_at = NOW() WHERE id = $1;

-- name: PurgeUser :exec
UPDATE users SET email = 'deleted-' || id::text || '@deleted.invalid', password_hash = NULL, display_name = NULL, updated_at = NOW() WHERE id = $1;
