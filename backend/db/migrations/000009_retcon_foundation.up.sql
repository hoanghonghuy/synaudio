-- Retcon + Account Purge Foundation
-- Phase 6 scope: retcon_requests, retcon_impacts, retcon_repair_tasks.

-- ============================================================
-- Retcon Requests
-- ============================================================

CREATE TABLE retcon_requests (
    id                            UUID PRIMARY KEY,
    story_id                      UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    target_chapter_id             UUID REFERENCES chapters (id) ON DELETE CASCADE,

    status                        TEXT NOT NULL DEFAULT 'DRAFT',
    impact_scope                  TEXT NOT NULL DEFAULT 'LOCAL',

    proposed_change               TEXT,
    reason                        TEXT NOT NULL,

    requested_by                  UUID REFERENCES users (id),
    approved_by                   UUID REFERENCES users (id),
    applied_by                    UUID REFERENCES users (id),

    base_official_canon_version_id UUID,
    workspace_branch_id           UUID,

    listener_impact               JSONB,

    created_at                    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_at                   TIMESTAMPTZ,
    applied_at                    TIMESTAMPTZ
);

CREATE INDEX idx_retcon_requests_story_id ON retcon_requests (story_id);
CREATE INDEX idx_retcon_requests_status ON retcon_requests (status);

-- ============================================================
-- Retcon Impacts
-- ============================================================

CREATE TABLE retcon_impacts (
    id               UUID PRIMARY KEY,
    retcon_request_id UUID NOT NULL REFERENCES retcon_requests (id) ON DELETE CASCADE,

    entity_type      TEXT NOT NULL,
    entity_id        UUID,

    impact_type      TEXT NOT NULL,
    detail           JSONB
);

CREATE INDEX idx_retcon_impacts_request_id ON retcon_impacts (retcon_request_id);

-- ============================================================
-- Retcon Repair Tasks
-- ============================================================

CREATE TABLE retcon_repair_tasks (
    id               UUID PRIMARY KEY,
    retcon_request_id UUID NOT NULL REFERENCES retcon_requests (id) ON DELETE CASCADE,

    task_type        TEXT NOT NULL,
    entity_type      TEXT,
    entity_id        UUID,

    status           TEXT NOT NULL DEFAULT 'PENDING',

    generation_run_id UUID,
    detail           JSONB,

    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at     TIMESTAMPTZ
);

CREATE INDEX idx_retcon_repair_tasks_request_id ON retcon_repair_tasks (retcon_request_id);
