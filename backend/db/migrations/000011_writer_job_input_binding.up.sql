-- Freeze the exact Chapter Plan revision used by each WRITER job.
-- A job-level binding is required because batch generation can place multiple
-- chapters (and therefore multiple plan revisions) inside one GenerationRun.

CREATE TABLE generation_job_writer_inputs (
    job_id           UUID PRIMARY KEY REFERENCES generation_jobs (id) ON DELETE CASCADE,
    chapter_id       UUID NOT NULL REFERENCES chapters (id) ON DELETE CASCADE,
    plan_revision_id UUID NOT NULL REFERENCES chapter_plan_revisions (id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_generation_job_writer_inputs_plan_revision
    ON generation_job_writer_inputs (plan_revision_id);

-- One WRITER business output is allowed for a frozen run/plan pair. If a worker
-- retries after persisting content but before recording the job output, the
-- existing revision is reused rather than producing an uncontrolled duplicate.
CREATE UNIQUE INDEX uq_writer_content_run_plan
    ON chapter_content_revisions (generation_run_id, plan_revision_id)
    WHERE generation_run_id IS NOT NULL
      AND plan_revision_id IS NOT NULL
      AND source_type = 'AI_GENERATED';
