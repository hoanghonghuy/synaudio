-- Generation + Content Revision Foundation
-- Phase 3 scope: GenerationRun, GenerationJob, JobAttempt, Content Revisions,
-- Content Approvals, Chapter Reviews, Content Classifications, Summaries.

-- ============================================================
-- Generation Runs
-- ============================================================

CREATE TABLE generation_runs (
    id                    UUID PRIMARY KEY,
    run_type              TEXT NOT NULL,
    story_id              UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    chapter_id            UUID REFERENCES chapters (id) ON DELETE CASCADE,
    status                TEXT NOT NULL DEFAULT 'PENDING',
    waiting_reason        TEXT,
    workflow_version      TEXT,
    priority              INTEGER NOT NULL DEFAULT 0,
    base_canon_version_id UUID,
    context_snapshot_id   UUID,
    requested_by          UUID REFERENCES users (id),
    idempotency_key       TEXT,
    started_at            TIMESTAMPTZ,
    completed_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (idempotency_key)
);

CREATE INDEX idx_generation_runs_story_id ON generation_runs (story_id);
CREATE INDEX idx_generation_runs_status ON generation_runs (status);

-- ============================================================
-- Generation Jobs
-- ============================================================

CREATE TABLE generation_jobs (
    id                UUID PRIMARY KEY,
    run_id            UUID NOT NULL REFERENCES generation_runs (id) ON DELETE CASCADE,
    job_type          TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'PENDING',
    priority          INTEGER NOT NULL DEFAULT 0,
    available_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    input_fingerprint TEXT,
    attempt_count     INTEGER NOT NULL DEFAULT 0,
    max_attempts      INTEGER NOT NULL DEFAULT 3,
    locked_by         TEXT,
    lock_expires_at   TIMESTAMPTZ,
    started_at        TIMESTAMPTZ,
    completed_at      TIMESTAMPTZ,
    last_error_class  TEXT,
    last_error_code   TEXT,
    output_ref        JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_generation_jobs_run_id ON generation_jobs (run_id);
CREATE INDEX idx_generation_jobs_status ON generation_jobs (status);

CREATE TABLE generation_job_dependencies (
    job_id           UUID NOT NULL REFERENCES generation_jobs (id) ON DELETE CASCADE,
    depends_on_job_id UUID NOT NULL REFERENCES generation_jobs (id) ON DELETE CASCADE,
    PRIMARY KEY (job_id, depends_on_job_id)
);

-- ============================================================
-- Generation Job Attempts
-- ============================================================

CREATE TABLE generation_job_attempts (
    id                UUID PRIMARY KEY,
    job_id            UUID NOT NULL REFERENCES generation_jobs (id) ON DELETE CASCADE,
    attempt_no        INTEGER NOT NULL,
    provider          TEXT,
    model             TEXT,
    status            TEXT NOT NULL DEFAULT 'RUNNING',
    error_class       TEXT,
    error_code        TEXT,
    safe_error_detail JSONB,
    usage             JSONB,
    latency_ms        INTEGER,
    started_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at      TIMESTAMPTZ,
    UNIQUE (job_id, attempt_no)
);

CREATE INDEX idx_generation_job_attempts_job_id ON generation_job_attempts (job_id);

-- ============================================================
-- Chapter Content Revisions
-- ============================================================

CREATE TABLE chapter_content_revisions (
    id                    UUID PRIMARY KEY,
    chapter_id            UUID NOT NULL REFERENCES chapters (id) ON DELETE CASCADE,
    revision_no           INTEGER NOT NULL,
    content_text          TEXT NOT NULL,
    source_type           TEXT NOT NULL DEFAULT 'AI_GENERATED',
    based_on_revision_id  UUID REFERENCES chapter_content_revisions (id),
    plan_revision_id      UUID REFERENCES chapter_plan_revisions (id),
    base_canon_version_id UUID,
    generation_run_id     UUID,
    retcon_request_id     UUID,
    status                TEXT NOT NULL DEFAULT 'CANDIDATE',
    created_by            UUID REFERENCES users (id),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (chapter_id, revision_no)
);

CREATE INDEX idx_chapter_content_revisions_chapter_id ON chapter_content_revisions (chapter_id);

-- ============================================================
-- Content Approvals (append-only)
-- ============================================================

CREATE TABLE content_approvals (
    id                 UUID PRIMARY KEY,
    chapter_id         UUID NOT NULL REFERENCES chapters (id) ON DELETE CASCADE,
    content_revision_id UUID NOT NULL REFERENCES chapter_content_revisions (id) ON DELETE CASCADE,
    approved_by        UUID REFERENCES users (id),
    approved_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    warnings_snapshot  JSONB,
    override_snapshot  JSONB
);

CREATE INDEX idx_content_approvals_chapter_id ON content_approvals (chapter_id);

-- ============================================================
-- Chapter Reviews
-- ============================================================

CREATE TABLE chapter_reviews (
    id                  UUID PRIMARY KEY,
    chapter_id          UUID NOT NULL REFERENCES chapters (id) ON DELETE CASCADE,
    content_revision_id UUID REFERENCES chapter_content_revisions (id) ON DELETE CASCADE,
    review_type         TEXT NOT NULL,
    canon_version_id    UUID,
    policy_version_id   UUID,
    outcome             TEXT NOT NULL,
    report              JSONB,
    generation_run_id   UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chapter_reviews_chapter_id ON chapter_reviews (chapter_id);

-- ============================================================
-- Content Classifications
-- ============================================================

CREATE TABLE content_classifications (
    id                  UUID PRIMARY KEY,
    content_revision_id UUID NOT NULL REFERENCES chapter_content_revisions (id) ON DELETE CASCADE,
    rating              TEXT,
    warnings            TEXT[],
    outcome             TEXT NOT NULL,
    policy_version_id   UUID,
    report              JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_content_classifications_revision_id ON content_classifications (content_revision_id);

-- ============================================================
-- Summaries
-- ============================================================

CREATE TABLE chapter_summaries (
    id                  UUID PRIMARY KEY,
    chapter_id          UUID NOT NULL REFERENCES chapters (id) ON DELETE CASCADE,
    content_revision_id UUID REFERENCES chapter_content_revisions (id) ON DELETE CASCADE,
    summary             TEXT NOT NULL,
    generation_run_id   UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chapter_summaries_chapter_id ON chapter_summaries (chapter_id);

CREATE TABLE story_summaries (
    id               UUID PRIMARY KEY,
    story_id         UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    scope_type       TEXT NOT NULL,
    scope_id         UUID,
    canon_version_id UUID,
    summary          TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_story_summaries_story_id ON story_summaries (story_id);
