-- ============================================================
-- WRITER job immutable input + durable output
-- ============================================================

-- name: CreateWriterGenerationJob :one
WITH frozen_input AS (
    SELECT c.id AS chapter_id, c.current_plan_revision_id AS plan_revision_id
    FROM chapters c
    JOIN chapter_plan_revisions p
      ON p.id = c.current_plan_revision_id
     AND p.chapter_id = c.id
    WHERE c.id = sqlc.arg('chapter_id')
      AND c.current_plan_revision_id IS NOT NULL
), created_job AS (
    INSERT INTO generation_jobs (
        id, run_id, job_type, status, priority, input_fingerprint,
        attempt_count, max_attempts
    )
    SELECT sqlc.arg('id'), sqlc.arg('run_id'), sqlc.arg('job_type'), sqlc.arg('status'),
           sqlc.arg('priority'), sqlc.arg('input_fingerprint'), sqlc.arg('attempt_count'),
           sqlc.arg('max_attempts')
    FROM frozen_input
    RETURNING id
), bound_input AS (
    INSERT INTO generation_job_writer_inputs (job_id, chapter_id, plan_revision_id)
    SELECT j.id, f.chapter_id, f.plan_revision_id
    FROM created_job j
    CROSS JOIN frozen_input f
    RETURNING job_id
)
SELECT g.*
FROM generation_jobs g
JOIN bound_input b ON b.job_id = g.id
WHERE g.id = sqlc.arg('id');

-- name: GetWriterJobInput :one
SELECT wi.job_id, wi.chapter_id, wi.plan_revision_id, p.plan, p.base_canon_version_id
FROM generation_job_writer_inputs wi
JOIN chapter_plan_revisions p
  ON p.id = wi.plan_revision_id
 AND p.chapter_id = wi.chapter_id
WHERE wi.job_id = $1;

-- name: GetWriterOutput :one
SELECT id, chapter_id, revision_no, content_text, source_type, based_on_revision_id,
       plan_revision_id, base_canon_version_id, generation_run_id, retcon_request_id,
       status, created_by, created_at
FROM chapter_content_revisions
WHERE generation_run_id = $1
  AND plan_revision_id = $2
  AND source_type = 'AI_GENERATED'
ORDER BY revision_no DESC
LIMIT 1;

-- name: UpdateWriterJobOutputRef :one
UPDATE generation_jobs
SET output_ref = $2
WHERE id = $1
RETURNING id, run_id, job_type, status, priority, available_at, input_fingerprint,
          attempt_count, max_attempts, locked_by, lock_expires_at, started_at,
          completed_at, last_error_class, last_error_code, output_ref, created_at;