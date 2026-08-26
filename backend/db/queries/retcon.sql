-- ============================================================
-- Retcon Requests
-- ============================================================

-- name: CreateRetconRequest :one
INSERT INTO retcon_requests (id, story_id, target_chapter_id, status, impact_scope,
                             proposed_change, reason, requested_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, story_id, target_chapter_id, status, impact_scope, proposed_change, reason,
          requested_by, approved_by, applied_by, base_official_canon_version_id,
          workspace_branch_id, listener_impact, created_at, approved_at, applied_at;

-- name: GetRetconRequest :one
SELECT id, story_id, target_chapter_id, status, impact_scope, proposed_change, reason,
       requested_by, approved_by, applied_by, base_official_canon_version_id,
       workspace_branch_id, listener_impact, created_at, approved_at, applied_at
FROM retcon_requests
WHERE id = $1;

-- name: ListRetconRequests :many
SELECT id, story_id, target_chapter_id, status, impact_scope, proposed_change, reason,
       requested_by, approved_by, applied_by, base_official_canon_version_id,
       workspace_branch_id, listener_impact, created_at, approved_at, applied_at
FROM retcon_requests
WHERE story_id = $1
ORDER BY created_at;

-- name: UpdateRetconRequest :one
UPDATE retcon_requests
SET status = $2,
    approved_by = $3,
    applied_by = $4,
    approved_at = CASE WHEN $2 = 'APPROVED' THEN NOW() ELSE approved_at END,
    applied_at = CASE WHEN $2 = 'APPLIED' THEN NOW() ELSE applied_at END
WHERE id = $1
RETURNING id, story_id, target_chapter_id, status, impact_scope, proposed_change, reason,
          requested_by, approved_by, applied_by, base_official_canon_version_id,
          workspace_branch_id, listener_impact, created_at, approved_at, applied_at;
