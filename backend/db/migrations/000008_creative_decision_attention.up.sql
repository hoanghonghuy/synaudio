-- Creative Decisions + Attention Queue Foundation
-- Phase 5 scope: CreativeDecision, CreativeDecisionOptions, AttentionItems.

-- ============================================================
-- Creative Decisions
-- ============================================================

CREATE TABLE creative_decisions (
    id                    UUID PRIMARY KEY,
    story_id              UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    chapter_id            UUID REFERENCES chapters (id) ON DELETE CASCADE,
    arc_id                UUID REFERENCES story_arcs (id) ON DELETE CASCADE,

    origin                TEXT NOT NULL DEFAULT 'AI',
    decision_type         TEXT NOT NULL,
    severity              TEXT NOT NULL DEFAULT 'SIGNIFICANT',
    status                TEXT NOT NULL DEFAULT 'PROPOSED',
    blocking_level        TEXT NOT NULL DEFAULT 'NON_BLOCKING',

    question              TEXT NOT NULL,
    context_summary       TEXT,

    recommended_option_id UUID,
    selected_option_id    UUID,
    custom_selected_text  TEXT,

    rejection_scope       TEXT,
    revisit_condition     JSONB,

    triggered_by_run_id   UUID,
    created_by            UUID REFERENCES users (id),
    selected_by           UUID REFERENCES users (id),

    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    selected_at           TIMESTAMPTZ,
    applied_at            TIMESTAMPTZ
);

CREATE INDEX idx_creative_decisions_story_id ON creative_decisions (story_id);
CREATE INDEX idx_creative_decisions_status ON creative_decisions (status);
CREATE INDEX idx_creative_decisions_severity ON creative_decisions (severity);

-- ============================================================
-- Creative Decision Options
-- ============================================================

CREATE TABLE creative_decision_options (
    id             UUID PRIMARY KEY,
    decision_id    UUID NOT NULL REFERENCES creative_decisions (id) ON DELETE CASCADE,
    ordinal        INTEGER NOT NULL,
    title          TEXT NOT NULL,
    description    TEXT,
    impact         JSONB,
    is_recommended BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (decision_id, ordinal)
);

CREATE INDEX idx_creative_decision_options_decision_id ON creative_decision_options (decision_id);

-- ============================================================
-- Attention Items (in-app attention center)
-- ============================================================

CREATE TABLE attention_items (
    id           UUID PRIMARY KEY,
    story_id     UUID REFERENCES stories (id) ON DELETE CASCADE,
    chapter_id   UUID REFERENCES chapters (id) ON DELETE CASCADE,
    priority     TEXT NOT NULL DEFAULT 'INFORMATIONAL',
    kind         TEXT NOT NULL,
    title        TEXT NOT NULL,
    detail       TEXT,
    action       TEXT,
    resolved     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at  TIMESTAMPTZ
);

CREATE INDEX idx_attention_items_story_id ON attention_items (story_id, resolved);
CREATE INDEX idx_attention_items_priority ON attention_items (priority, resolved);
