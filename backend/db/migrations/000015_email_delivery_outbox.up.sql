CREATE TABLE email_delivery_outbox (
    id                 UUID PRIMARY KEY,
    purpose            TEXT NOT NULL CHECK (purpose IN ('EMAIL_VERIFICATION', 'PASSWORD_RESET')),
    recipient_email    TEXT NOT NULL,
    encrypted_payload  BYTEA NOT NULL,
    status             TEXT NOT NULL DEFAULT 'PENDING'
                       CHECK (status IN ('PENDING', 'DELIVERING', 'DELIVERED', 'DEAD_LETTER')),
    attempt_count      INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    max_attempts       INTEGER NOT NULL DEFAULT 8 CHECK (max_attempts > 0),
    available_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_at          TIMESTAMPTZ,
    last_error         TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_delivery_outbox_runnable
    ON email_delivery_outbox (status, available_at, created_at)
    WHERE status IN ('PENDING', 'DELIVERING');

CREATE INDEX idx_email_delivery_outbox_dead_letter
    ON email_delivery_outbox (updated_at DESC)
    WHERE status = 'DEAD_LETTER';
