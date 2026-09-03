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
WHERE ($1::uuid IS NULL OR actor_user_id = $1)
  AND ($2::text IS NULL OR action = $2)
  AND ($3::text IS NULL OR resource_type = $3)
  AND ($4::text IS NULL OR resource_id = $4)
  AND ($5::uuid IS NULL OR story_id = $5)
  AND ($6::uuid IS NULL OR chapter_id = $6)
  AND ($7::uuid IS NULL OR generation_run_id = $7)
  AND ($8::text IS NULL OR correlation_id = $8)
  AND ($9::text IS NULL OR result = $9)
  AND ($10::timestamptz IS NULL OR created_at >= $10)
  AND ($11::timestamptz IS NULL OR created_at <= $11)
ORDER BY created_at DESC, id DESC
LIMIT $12;
