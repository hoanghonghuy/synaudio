-- Foundation placeholder query used to verify sqlc generation wiring.
-- name: PingBootstrap :one
SELECT id, created_at
FROM schema_bootstrap
WHERE id = TRUE;
