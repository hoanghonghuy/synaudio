-- ============================================================
-- Generation Runs
-- ============================================================

-- name: CreateGenerationRun :one
INSERT INTO generation_runs (id, run_type, story_id, chapter_id, status, waiting_reason,
                             workflow_version, priority, base_canon_version_id,
                             context_snapshot_id, requested_by, idempotency_key)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, run_type, story_id, chapter_id, status, waiting_reason, workflow_version,
          priority, base_canon_version_id, context_snapshot_id, requested_by,
          idempotency_key, started_at, completed_at, created_at;

-- name: GetGenerationRun :one
SELECT id, run_type, story_id, chapter_id, status, waiting_reason, workflow_version,
       priority, base_canon_version_id, context_snapshot_id, requested_by,
       idempotency_key, started_at, completed_at, created_at
FROM generation_runs
WHERE id = $1;

-- ============================================================
-- Generation Jobs
-- ============================================================

-- name: CreateGenerationJob :one
INSERT INTO generation_jobs (id, run_id, job_type, status, priority, available_at,
                             input_fingerprint, attempt_count, max_attempts)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, run_id, job_type, status, priority, available_at, input_fingerprint,
          attempt_count, max_attempts, locked_by, lock_expires_at, started_at,
          completed_at, last_error_class, last_error_code, output_ref, created_at;

-- name: NextAttemptNo :one
SELECT COALESCE(MAX(attempt_no), 0) + 1
FROM generation_job_attempts
WHERE job_id = $1;

-- name: CreateJobAttempt :one
INSERT INTO generation_job_attempts (id, job_id, attempt_no, provider, model, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, job_id, attempt_no, provider, model, status, error_class, error_code,
          safe_error_detail, usage, latency_ms, started_at, completed_at;

-- ============================================================
-- Chapter Content Revisions
-- ============================================================

-- name: NextContentRevision :one
SELECT COALESCE(MAX(revision_no), 0) + 1
FROM chapter_content_revisions
WHERE chapter_id = $1;

-- name: CreateContentRevision :one
INSERT INTO chapter_content_revisions (id, chapter_id, revision_no, content_text, source_type,
                                       based_on_revision_id, plan_revision_id,
                                       base_canon_version_id, generation_run_id,
                                       retcon_request_id, status, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, chapter_id, revision_no, content_text, source_type, based_on_revision_id,
          plan_revision_id, base_canon_version_id, generation_run_id, retcon_request_id,
          status, created_by, created_at;

-- name: GetContentRevision :one
SELECT id, chapter_id, revision_no, content_text, source_type, based_on_revision_id,
       plan_revision_id, base_canon_version_id, generation_run_id, retcon_request_id,
       status, created_by, created_at
FROM chapter_content_revisions
WHERE id = $1;

-- name: ListContentRevisions :many
SELECT id, chapter_id, revision_no, content_text, source_type, based_on_revision_id,
       plan_revision_id, base_canon_version_id, generation_run_id, retcon_request_id,
       status, created_by, created_at
FROM chapter_content_revisions
WHERE chapter_id = $1
ORDER BY revision_no;

-- name: CreateContentApproval :one
INSERT INTO content_approvals (id, chapter_id, content_revision_id, approved_by,
                               warnings_snapshot, override_snapshot)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, chapter_id, content_revision_id, approved_by, approved_at,
          warnings_snapshot, override_snapshot;

-- ============================================================
-- Chapter Reviews
-- ============================================================

-- name: CreateChapterReview :one
INSERT INTO chapter_reviews (id, chapter_id, content_revision_id, review_type,
                             canon_version_id, policy_version_id, outcome, report,
                             generation_run_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, chapter_id, content_revision_id, review_type, canon_version_id,
          policy_version_id, outcome, report, generation_run_id, created_at;

-- name: ListChapterReviews :many
SELECT id, chapter_id, content_revision_id, review_type, canon_version_id,
       policy_version_id, outcome, report, generation_run_id, created_at
FROM chapter_reviews
WHERE chapter_id = $1
ORDER BY created_at;
