-- name: CreateAuditEvent :one
INSERT INTO audit_events (
    id, actor_user_id, actor_type, action, resource_type, resource_id,
    story_id, chapter_id, result, correlation_id, request_id,
    generation_run_id, provenance, metadata
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING id, actor_user_id, actor_type, action, resource_type, resource_id,
          story_id, chapter_id, result, correlation_id, request_id,
          generation_run_id, provenance, metadata, created_at;

-- name: GetAuditEvent :one
SELECT id, actor_user_id, actor_type, action, resource_type, resource_id,
       story_id, chapter_id, result, correlation_id, request_id,
       generation_run_id, provenance, metadata, created_at
FROM audit_events
WHERE id = $1;

-- name: ListAuditEvents :many
SELECT id, actor_user_id, actor_type, action, resource_type, resource_id,
       story_id, chapter_id, result, correlation_id, request_id,
       generation_run_id, provenance, metadata, created_at
FROM audit_events
WHERE (sqlc.narg('actor_user_id')::uuid IS NULL OR actor_user_id = sqlc.narg('actor_user_id'))
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action'))
  AND (sqlc.narg('resource_type')::text IS NULL OR resource_type = sqlc.narg('resource_type'))
  AND (sqlc.narg('resource_id')::text IS NULL OR resource_id = sqlc.narg('resource_id'))
  AND (sqlc.narg('story_id')::uuid IS NULL OR story_id = sqlc.narg('story_id'))
  AND (sqlc.narg('chapter_id')::uuid IS NULL OR chapter_id = sqlc.narg('chapter_id'))
  AND (sqlc.narg('generation_run_id')::uuid IS NULL OR generation_run_id = sqlc.narg('generation_run_id'))
  AND (sqlc.narg('correlation_id')::text IS NULL OR correlation_id = sqlc.narg('correlation_id'))
  AND (sqlc.narg('result')::text IS NULL OR result = sqlc.narg('result'))
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from'))
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at <= sqlc.narg('created_to'))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit');