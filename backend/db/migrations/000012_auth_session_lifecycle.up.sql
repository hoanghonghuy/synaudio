-- V1 authentication/session lifecycle hardening.
-- Refresh tokens are random opaque credentials and hashes are unique at rest.

CREATE UNIQUE INDEX uq_user_sessions_refresh_token_hash
    ON user_sessions (refresh_token_hash);

CREATE INDEX idx_user_sessions_active_by_user
    ON user_sessions (user_id, revoked_at, expires_at);
