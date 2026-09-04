CREATE TABLE account_deletion_recovery_tokens (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, token_hash)
);

CREATE INDEX idx_account_deletion_recovery_active
    ON account_deletion_recovery_tokens (user_id, expires_at)
    WHERE used_at IS NULL;
