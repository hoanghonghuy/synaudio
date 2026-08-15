-- Planning + Canon Foundation
-- Phase 2 scope: Story Bible, Ending Plan, Arcs, Characters, Character States,
-- Chapters, Chapter Plans, StoryFacts, PlotThreads, Canon branches/versions,
-- ContextSnapshots.

-- ============================================================
-- Story Bible (versioned)
-- ============================================================

CREATE TABLE story_bible_versions (
    id                 UUID PRIMARY KEY,
    story_id           UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    version_no         INTEGER NOT NULL,
    content            JSONB NOT NULL,
    based_on_version_id UUID REFERENCES story_bible_versions (id),
    created_by         UUID REFERENCES users (id),
    generation_run_id  UUID,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (story_id, version_no)
);

CREATE INDEX idx_story_bible_versions_story_id ON story_bible_versions (story_id);

-- ============================================================
-- Ending Plan (versioned)
-- ============================================================

CREATE TABLE story_ending_plan_versions (
    id                 UUID PRIMARY KEY,
    story_id           UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    version_no         INTEGER NOT NULL,
    content            JSONB NOT NULL,
    based_on_version_id UUID REFERENCES story_ending_plan_versions (id),
    created_by         UUID REFERENCES users (id),
    generation_run_id  UUID,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (story_id, version_no)
);

CREATE INDEX idx_story_ending_plan_versions_story_id ON story_ending_plan_versions (story_id);

-- ============================================================
-- Story Arcs (stable identity + versioned content)
-- ============================================================

CREATE TABLE story_arcs (
    id                 UUID PRIMARY KEY,
    story_id           UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    ordinal            INTEGER NOT NULL,
    status             TEXT NOT NULL DEFAULT 'PLANNED',
    current_version_id UUID,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (story_id, ordinal)
);

CREATE INDEX idx_story_arcs_story_id ON story_arcs (story_id);

CREATE TABLE story_arc_versions (
    id                  UUID PRIMARY KEY,
    arc_id              UUID NOT NULL REFERENCES story_arcs (id) ON DELETE CASCADE,
    version_no          INTEGER NOT NULL,
    content             JSONB NOT NULL,
    base_canon_version_id UUID,
    generation_run_id   UUID,
    created_by          UUID REFERENCES users (id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (arc_id, version_no)
);

CREATE INDEX idx_story_arc_versions_arc_id ON story_arc_versions (arc_id);

-- Migration-safe circular pointer: story_arcs.current_version_id
ALTER TABLE story_arcs
    ADD CONSTRAINT fk_story_arcs_current_version
    FOREIGN KEY (current_version_id) REFERENCES story_arc_versions (id);

-- ============================================================
-- Characters (stable identity + versioned profile + immutable state)
-- ============================================================

CREATE TABLE characters (
    id                        UUID PRIMARY KEY,
    story_id                  UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    canonical_name            TEXT NOT NULL,
    importance                TEXT NOT NULL DEFAULT 'MINOR',
    current_profile_version_id UUID,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_characters_story_id ON characters (story_id);

CREATE TABLE character_profile_versions (
    id                  UUID PRIMARY KEY,
    character_id        UUID NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    version_no          INTEGER NOT NULL,
    profile             JSONB NOT NULL,
    base_canon_version_id UUID,
    created_by          UUID REFERENCES users (id),
    generation_run_id   UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (character_id, version_no)
);

CREATE INDEX idx_character_profile_versions_character_id ON character_profile_versions (character_id);

-- Migration-safe circular pointer: characters.current_profile_version_id
ALTER TABLE characters
    ADD CONSTRAINT fk_characters_current_profile_version
    FOREIGN KEY (current_profile_version_id) REFERENCES character_profile_versions (id);

CREATE TABLE character_state_versions (
    id                        UUID PRIMARY KEY,
    character_id              UUID NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    canon_version_id          UUID,
    state                     JSONB NOT NULL,
    source_chapter_id         UUID,
    source_content_revision_id UUID,
    generation_run_id         UUID,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_character_state_versions_character_id ON character_state_versions (character_id);

-- ============================================================
-- Chapters
-- ============================================================

CREATE TABLE chapters (
    id                          UUID PRIMARY KEY,
    story_id                    UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    chapter_number              INTEGER NOT NULL,
    title                       TEXT,
    status                      TEXT NOT NULL DEFAULT 'DRAFT',
    arc_id                      UUID REFERENCES story_arcs (id),
    current_plan_revision_id    UUID,
    current_content_revision_id UUID,
    current_narration_revision_id UUID,
    current_audio_asset_id      UUID,
    official_canon_version_id   UUID,
    published_at                TIMESTAMPTZ,
    archived_at                 TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (story_id, chapter_number)
);

CREATE INDEX idx_chapters_story_id ON chapters (story_id);

-- ============================================================
-- Chapter Plan Revisions
-- ============================================================

CREATE TABLE chapter_plan_revisions (
    id                  UUID PRIMARY KEY,
    chapter_id          UUID NOT NULL REFERENCES chapters (id) ON DELETE CASCADE,
    revision_no         INTEGER NOT NULL,
    plan                JSONB NOT NULL,
    base_canon_version_id UUID,
    arc_version_id      UUID REFERENCES story_arc_versions (id),
    source_type         TEXT NOT NULL DEFAULT 'AI_GENERATED',
    generation_run_id   UUID,
    created_by          UUID REFERENCES users (id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (chapter_id, revision_no)
);

CREATE INDEX idx_chapter_plan_revisions_chapter_id ON chapter_plan_revisions (chapter_id);

-- ============================================================
-- StoryFacts
-- ============================================================

CREATE TABLE story_facts (
    id                          UUID PRIMARY KEY,
    story_id                    UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    subject_type                TEXT,
    subject_id                  UUID,
    fact_type                   TEXT NOT NULL,
    value                       JSONB NOT NULL,
    importance                  TEXT NOT NULL DEFAULT 'NORMAL',
    status                      TEXT NOT NULL DEFAULT 'ACTIVE',
    valid_from_canon_version_id UUID,
    invalidated_at_canon_version_id UUID,
    supersedes_fact_id          UUID REFERENCES story_facts (id),
    source_chapter_id           UUID,
    source_content_revision_id  UUID,
    generation_run_id           UUID,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_story_facts_story_status ON story_facts (story_id, status);
CREATE INDEX idx_story_facts_subject ON story_facts (story_id, subject_type, subject_id);
CREATE INDEX idx_story_facts_type ON story_facts (story_id, fact_type);

-- ============================================================
-- PlotThreads
-- ============================================================

CREATE TABLE plot_threads (
    id                      UUID PRIMARY KEY,
    story_id                UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    title                   TEXT NOT NULL,
    summary                 TEXT,
    importance              TEXT NOT NULL DEFAULT 'NORMAL',
    status                  TEXT NOT NULL DEFAULT 'OPEN',
    opened_chapter_id       UUID,
    resolved_chapter_id     UUID,
    last_advanced_chapter_id UUID,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plot_threads_story_id ON plot_threads (story_id);

CREATE TABLE plot_thread_events (
    id              UUID PRIMARY KEY,
    plot_thread_id  UUID NOT NULL REFERENCES plot_threads (id) ON DELETE CASCADE,
    canon_version_id UUID,
    chapter_id      UUID,
    event_type      TEXT NOT NULL,
    detail          JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plot_thread_events_thread_id ON plot_thread_events (plot_thread_id);

-- ============================================================
-- Canon branches + versions + change items
-- ============================================================

CREATE TABLE canon_branches (
    id                UUID PRIMARY KEY,
    story_id          UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    type              TEXT NOT NULL DEFAULT 'OFFICIAL',
    status            TEXT NOT NULL DEFAULT 'ACTIVE',
    base_version_id   UUID,
    generation_run_id UUID,
    retcon_request_id UUID,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_canon_branches_story_id ON canon_branches (story_id);

CREATE TABLE canon_versions (
    id                          UUID PRIMARY KEY,
    story_id                    UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    branch_id                   UUID NOT NULL REFERENCES canon_branches (id) ON DELETE CASCADE,
    sequence_no                 INTEGER NOT NULL,
    parent_version_id           UUID REFERENCES canon_versions (id),
    source_chapter_id           UUID,
    source_content_revision_id  UUID,
    source_provisional_version_id UUID,
    generation_run_id           UUID,
    retcon_request_id           UUID,
    status                      TEXT NOT NULL DEFAULT 'DRAFT',
    committed_by                UUID REFERENCES users (id),
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (branch_id, sequence_no)
);

CREATE INDEX idx_canon_versions_story_id ON canon_versions (story_id);
CREATE INDEX idx_canon_versions_branch_id ON canon_versions (branch_id);

CREATE TABLE canon_change_items (
    id              UUID PRIMARY KEY,
    canon_version_id UUID NOT NULL REFERENCES canon_versions (id) ON DELETE CASCADE,
    entity_type     TEXT NOT NULL,
    entity_id       UUID,
    change_type     TEXT NOT NULL,
    metadata        JSONB
);

CREATE INDEX idx_canon_change_items_version_id ON canon_change_items (canon_version_id);

-- ============================================================
-- ContextSnapshots (immutable)
-- ============================================================

CREATE TABLE context_snapshots (
    id                        UUID PRIMARY KEY,
    run_id                    UUID,
    story_id                  UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    chapter_id                UUID,
    canon_version_id          UUID,
    bible_version_id          UUID,
    ending_plan_version_id    UUID,
    arc_version_id            UUID,
    content_profile_version_id UUID,
    prompt_version            TEXT,
    workflow_version          TEXT,
    provider                  TEXT,
    model                     TEXT,
    included_refs             JSONB,
    historical_hits           JSONB,
    admin_instruction         TEXT,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_context_snapshots_story_id ON context_snapshots (story_id);
