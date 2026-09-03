CREATE TABLE audit_delivery_outbox (
    id            UUID PRIMARY KEY,
    event         JSONB NOT NULL,
    status        TEXT NOT NULL DEFAULT 'PENDING'
                  CHECK (status IN ('PENDING', 'DELIVERING', 'DELIVERED', 'DEAD_LETTER')),
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    max_attempts  INTEGER NOT NULL DEFAULT 8 CHECK (max_attempts > 0),
    available_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_at     TIMESTAMPTZ,
    last_error    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_delivery_outbox_runnable
    ON audit_delivery_outbox (status, available_at, created_at)
    WHERE status IN ('PENDING', 'DELIVERING');

CREATE INDEX idx_audit_delivery_outbox_dead_letter
    ON audit_delivery_outbox (updated_at DESC)
    WHERE status = 'DEAD_LETTER';
