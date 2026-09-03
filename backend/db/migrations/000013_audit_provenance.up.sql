-- Append-only platform audit/provenance events.
-- Source identifiers are intentionally not foreign keys: audit history must
-- survive source-row lifecycle changes without being updated or nulled.

CREATE TABLE audit_events (
    id                UUID PRIMARY KEY,
    actor_user_id     UUID,
    actor_type        TEXT NOT NULL CHECK (actor_type IN ('USER', 'SYSTEM', 'AI', 'ANONYMOUS')),
    action            TEXT NOT NULL,
    resource_type     TEXT,
    resource_id       TEXT,
    story_id          UUID,
    chapter_id        UUID,
    result            TEXT NOT NULL CHECK (result IN ('SUCCEEDED', 'FAILED', 'DENIED')),
    correlation_id    TEXT,
    request_id        TEXT,
    generation_run_id UUID,
    provenance        JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata          JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_created_at ON audit_events (created_at DESC);
CREATE INDEX idx_audit_events_story_time ON audit_events (story_id, created_at DESC) WHERE story_id IS NOT NULL;
CREATE INDEX idx_audit_events_chapter_time ON audit_events (chapter_id, created_at DESC) WHERE chapter_id IS NOT NULL;
CREATE INDEX idx_audit_events_actor_time ON audit_events (actor_user_id, created_at DESC) WHERE actor_user_id IS NOT NULL;
CREATE INDEX idx_audit_events_action_time ON audit_events (action, created_at DESC);
CREATE INDEX idx_audit_events_resource ON audit_events (resource_type, resource_id, created_at DESC);
CREATE INDEX idx_audit_events_run ON audit_events (generation_run_id, created_at DESC) WHERE generation_run_id IS NOT NULL;
CREATE INDEX idx_audit_events_correlation ON audit_events (correlation_id, created_at DESC) WHERE correlation_id IS NOT NULL;

CREATE FUNCTION synaudio_reject_audit_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_events is append-only';
END;
$$;

CREATE TRIGGER audit_events_append_only
BEFORE UPDATE OR DELETE ON audit_events
FOR EACH ROW EXECUTE FUNCTION synaudio_reject_audit_mutation();
