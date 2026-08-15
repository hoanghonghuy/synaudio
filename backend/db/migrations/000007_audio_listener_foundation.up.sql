-- Audio + Publishing + Listener Foundation
-- Phase 4 scope: narration revisions, TTS segments, audio assets,
-- favorites, playback sessions, listening progress.

-- ============================================================
-- Narration Revisions
-- ============================================================

CREATE TABLE narration_revisions (
    id                        UUID PRIMARY KEY,
    chapter_id                UUID NOT NULL REFERENCES chapters (id) ON DELETE CASCADE,
    revision_no               INTEGER NOT NULL,
    source_content_revision_id UUID REFERENCES chapter_content_revisions (id),
    voice_id                  TEXT,
    script                    TEXT,
    status                    TEXT NOT NULL DEFAULT 'DRAFT',
    generation_run_id         UUID,
    created_by                UUID REFERENCES users (id),
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (chapter_id, revision_no)
);

CREATE INDEX idx_narration_revisions_chapter_id ON narration_revisions (chapter_id);

-- ============================================================
-- TTS Segments
-- ============================================================

CREATE TABLE tts_segments (
    id                    UUID PRIMARY KEY,
    narration_revision_id UUID NOT NULL REFERENCES narration_revisions (id) ON DELETE CASCADE,
    segment_no            INTEGER NOT NULL,
    text                  TEXT NOT NULL,
    direction             JSONB,
    status                TEXT NOT NULL DEFAULT 'PENDING',
    provider              TEXT,
    model                 TEXT,
    voice_id              TEXT,
    duration_ms           INTEGER,
    temp_storage_key      TEXT,
    generation_job_id     UUID,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (narration_revision_id, segment_no)
);

CREATE INDEX idx_tts_segments_narration_id ON tts_segments (narration_revision_id);

-- ============================================================
-- Audio Assets
-- ============================================================

CREATE TABLE audio_assets (
    id                          UUID PRIMARY KEY,
    chapter_id                  UUID NOT NULL REFERENCES chapters (id) ON DELETE CASCADE,
    version_no                  INTEGER NOT NULL,
    source_narration_revision_id UUID REFERENCES narration_revisions (id),
    status                      TEXT NOT NULL DEFAULT 'PENDING',
    storage_key                 TEXT,
    mime_type                   TEXT,
    size_bytes                  BIGINT,
    duration_ms                 INTEGER,
    bitrate_kbps                INTEGER,
    checksum                    TEXT,
    is_active                   BOOLEAN NOT NULL DEFAULT FALSE,
    generation_run_id           UUID,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (chapter_id, version_no)
);

CREATE INDEX idx_audio_assets_chapter_id ON audio_assets (chapter_id);
CREATE UNIQUE INDEX idx_audio_assets_active ON audio_assets (chapter_id) WHERE is_active = true;

-- ============================================================
-- Favorites
-- ============================================================

CREATE TABLE favorites (
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    story_id   UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, story_id)
);

CREATE INDEX idx_favorites_user_id ON favorites (user_id, created_at DESC);

-- ============================================================
-- Playback Sessions
-- ============================================================

CREATE TABLE playback_sessions (
    id                 UUID PRIMARY KEY,
    user_id            UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    chapter_id         UUID NOT NULL REFERENCES chapters (id) ON DELETE CASCADE,
    audio_asset_id     UUID REFERENCES audio_assets (id),
    client_instance_id TEXT,
    started_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_event_at      TIMESTAMPTZ,
    ended_at           TIMESTAMPTZ
);

CREATE INDEX idx_playback_sessions_user_id ON playback_sessions (user_id);

-- ============================================================
-- Listening Progress
-- ============================================================

CREATE TABLE listening_progress (
    user_id                 UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    chapter_id              UUID NOT NULL REFERENCES chapters (id) ON DELETE CASCADE,
    position_ms             BIGINT NOT NULL DEFAULT 0,
    completed_at            TIMESTAMPTZ,
    last_audio_asset_id     UUID,
    last_playback_session_id UUID,
    version                 BIGINT NOT NULL DEFAULT 0,
    last_listened_at        TIMESTAMPTZ,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, chapter_id)
);

CREATE INDEX idx_listening_progress_user_id ON listening_progress (user_id, last_listened_at DESC);
