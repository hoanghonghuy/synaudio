-- name: EnqueueAuditIntent :exec
INSERT INTO audit_delivery_outbox (id, event)
VALUES ($1, $2)
ON CONFLICT (id) DO NOTHING;

-- name: ClaimAuditIntents :many
WITH candidates AS (
    SELECT id
    FROM audit_delivery_outbox
    WHERE (
        status = 'PENDING' AND available_at <= NOW()
    ) OR (
        status = 'DELIVERING' AND locked_at <= NOW() - INTERVAL '5 minutes'
    )
    ORDER BY available_at ASC, created_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT $1
)
UPDATE audit_delivery_outbox o
SET status = 'DELIVERING',
    attempt_count = o.attempt_count + 1,
    locked_at = NOW(),
    updated_at = NOW()
FROM candidates c
WHERE o.id = c.id
RETURNING o.id, o.event, o.status, o.attempt_count, o.max_attempts,
          o.available_at, o.locked_at, o.last_error, o.created_at, o.updated_at;

-- name: MarkAuditIntentDelivered :exec
UPDATE audit_delivery_outbox
SET status = 'DELIVERED', locked_at = NULL, last_error = NULL, updated_at = NOW()
WHERE id = $1;

-- name: MarkAuditIntentFailed :one
UPDATE audit_delivery_outbox
SET status = $2,
    available_at = $3,
    locked_at = NULL,
    last_error = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING id, event, status, attempt_count, max_attempts,
          available_at, locked_at, last_error, created_at, updated_at;
