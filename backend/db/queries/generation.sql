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

-- name: ClaimNextJob :one
UPDATE generation_jobs
SET status = 'RUNNING',
    locked_by = $1,
    lock_expires_at = NOW() + INTERVAL '5 minutes',
    started_at = NOW(),
    attempt_count = attempt_count + 1
WHERE id = (
    SELECT id
    FROM generation_jobs
    WHERE status = 'PENDING'
      AND available_at <= NOW()
    ORDER BY priority DESC, created_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, run_id, job_type, status, priority, available_at, input_fingerprint,
          attempt_count, max_attempts, locked_by, lock_expires_at, started_at,
          completed_at, last_error_class, last_error_code, output_ref, created_at;

-- name: UpdateJobStatus :one
UPDATE generation_jobs
SET status = $2,
    last_error_class = $3,
    last_error_code = $4,
    completed_at = CASE WHEN $2 IN ('SUCCEEDED', 'FAILED', 'CANCELLED') THEN NOW() ELSE completed_at END,
    locked_by = NULL,
    lock_expires_at = NULL
WHERE id = $1
RETURNING id, run_id, job_type, status, priority, available_at, input_fingerprint,
          attempt_count, max_attempts, locked_by, lock_expires_at, started_at,
          completed_at, last_error_class, last_error_code, output_ref, created_at;

-- name: UpdateJobAttemptStatus :one
UPDATE generation_job_attempts
SET status = $2,
    error_class = $3,
    error_code = $4,
    completed_at = CASE WHEN $2 IN ('SUCCEEDED', 'FAILED') THEN NOW() ELSE completed_at END
WHERE id = $1
RETURNING id, job_id, attempt_no, provider, model, status, error_class, error_code,
          safe_error_detail, usage, latency_ms, started_at, completed_at;

-- name: ReclaimStaleJobs :many
UPDATE generation_jobs
SET status = 'PENDING',
    locked_by = NULL,
    lock_expires_at = NULL
WHERE status = 'RUNNING'
  AND lock_expires_at < NOW()
RETURNING id, run_id, job_type, status, priority, available_at, input_fingerprint,
          attempt_count, max_attempts, locked_by, lock_expires_at, started_at,
          completed_at, last_error_class, last_error_code, output_ref, created_at;

-- name: CancelJob :one
UPDATE generation_jobs
SET status = 'CANCELLED',
    completed_at = NOW(),
    locked_by = NULL,
    lock_expires_at = NULL
WHERE id = $1
RETURNING id, run_id, job_type, status, priority, available_at, input_fingerprint,
          attempt_count, max_attempts, locked_by, lock_expires_at, started_at,
          completed_at, last_error_class, last_error_code, output_ref, created_at;

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
