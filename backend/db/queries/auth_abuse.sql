-- name: IncrementAuthAbuseCounter :one
INSERT INTO auth_abuse_counters (scope, key_hash, window_start, count, updated_at)
VALUES ($1, $2, $3, 1, NOW())
ON CONFLICT (scope, key_hash, window_start)
DO UPDATE SET
    count = auth_abuse_counters.count + 1,
    updated_at = NOW()
RETURNING count;
