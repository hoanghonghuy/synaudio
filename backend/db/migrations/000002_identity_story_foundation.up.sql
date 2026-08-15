-- Identity + Story Foundation
-- Phase 1 scope: users, roles, permissions, sessions, MFA, tokens,
-- genres, stories, story assets, generation policy, workflow settings.

CREATE EXTENSION IF NOT EXISTS citext;

-- ============================================================
-- Identity
-- ============================================================

CREATE TABLE users (
    id                UUID PRIMARY KEY,
    email             CITEXT NOT NULL UNIQUE,
    password_hash     TEXT,
    display_name      TEXT,
    status            TEXT NOT NULL DEFAULT 'ACTIVE'
                      CHECK (status IN ('ACTIVE', 'SUSPENDED', 'DEACTIVATED')),
    email_verified_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deactivated_at    TIMESTAMPTZ
);

CREATE TABLE roles (
    id   UUID PRIMARY KEY,
    code TEXT NOT NULL UNIQUE
);

CREATE TABLE permissions (
    id   UUID PRIMARY KEY,
    code TEXT NOT NULL UNIQUE
);

CREATE TABLE role_permissions (
    role_id       UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    granted_by UUID REFERENCES users (id),
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, role_id)
);

-- ============================================================
-- Auth
-- ============================================================

CREATE TABLE user_sessions (
    id                 UUID PRIMARY KEY,
    user_id            UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    refresh_token_hash TEXT NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at       TIMESTAMPTZ,
    expires_at         TIMESTAMPTZ NOT NULL,
    revoked_at         TIMESTAMPTZ,
    mfa_verified_at    TIMESTAMPTZ,
    recent_auth_at     TIMESTAMPTZ,
    user_agent_summary TEXT,
    safe_ip_metadata   TEXT
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions (user_id);

CREATE TABLE user_mfa_methods (
    id               UUID PRIMARY KEY,
    user_id          UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type             TEXT NOT NULL,
    encrypted_secret TEXT NOT NULL,
    confirmed_at     TIMESTAMPTZ,
    disabled_at      TIMESTAMPTZ
);

CREATE INDEX idx_user_mfa_methods_user_id ON user_mfa_methods (user_id);

CREATE TABLE user_mfa_recovery_codes (
    id        UUID PRIMARY KEY,
    user_id   UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    used_at   TIMESTAMPTZ
);

CREATE INDEX idx_user_mfa_recovery_codes_user_id ON user_mfa_recovery_codes (user_id);

CREATE TABLE email_verification_tokens (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_verification_tokens_user_id ON email_verification_tokens (user_id);

CREATE TABLE password_reset_tokens (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens (user_id);

CREATE TABLE account_deletion_requests (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    purge_after  TIMESTAMPTZ NOT NULL,
    cancelled_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_account_deletion_requests_user_id ON account_deletion_requests (user_id);

-- ============================================================
-- Story foundation
-- ============================================================

CREATE TABLE genres (
    id   UUID PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE stories (
    id                                UUID PRIMARY KEY,
    slug                              TEXT NOT NULL UNIQUE,
    title                             TEXT NOT NULL,
    description                       TEXT,
    status                            TEXT NOT NULL DEFAULT 'DRAFT'
                                      CHECK (status IN ('DRAFT', 'ACTIVE', 'COMPLETED', 'ARCHIVED')),
    visibility                        TEXT NOT NULL DEFAULT 'PRIVATE'
                                      CHECK (visibility IN ('PRIVATE', 'PUBLIC')),
    planning_mode                     TEXT NOT NULL DEFAULT 'FINITE'
                                      CHECK (planning_mode IN ('FINITE', 'OPEN_ENDED')),
    planning_phase                    TEXT NOT NULL DEFAULT 'ONGOING'
                                      CHECK (planning_phase IN ('ONGOING', 'CLOSING', 'FINAL_ARC', 'COMPLETED')),
    public_rating                     TEXT,
    public_warnings                   TEXT[] NOT NULL DEFAULT '{}',
    cover_asset_id                    UUID,
    current_story_bible_version_id    UUID,
    current_ending_plan_version_id    UUID,
    current_content_profile_version_id UUID,
    current_official_canon_version_id UUID,
    public_since                      TIMESTAMPTZ,
    last_published_at                 TIMESTAMPTZ,
    status_before_archive             TEXT,
    created_by                        UUID REFERENCES users (id),
    created_at                        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at                       TIMESTAMPTZ
);

CREATE INDEX idx_stories_status ON stories (status);
CREATE INDEX idx_stories_visibility ON stories (visibility);

CREATE TABLE story_genres (
    story_id UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    genre_id UUID NOT NULL REFERENCES genres (id) ON DELETE CASCADE,
    PRIMARY KEY (story_id, genre_id)
);

CREATE TABLE story_assets (
    id            UUID PRIMARY KEY,
    story_id      UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    type          TEXT NOT NULL,
    storage_key   TEXT NOT NULL,
    mime_type     TEXT,
    size_bytes    BIGINT,
    checksum      TEXT,
    rights_status TEXT,
    status        TEXT NOT NULL DEFAULT 'PENDING',
    created_by    UUID REFERENCES users (id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_story_assets_story_id ON story_assets (story_id);

-- cover_asset_id FK added after story_assets exists (migration-safe circular pointer)
ALTER TABLE stories
    ADD CONSTRAINT fk_stories_cover_asset
    FOREIGN KEY (cover_asset_id) REFERENCES story_assets (id);

CREATE TABLE story_generation_policies (
    story_id                   UUID PRIMARY KEY REFERENCES stories (id) ON DELETE CASCADE,
    minimum_audio_duration_sec INTEGER NOT NULL,
    target_audio_duration_sec  INTEGER NOT NULL,
    content_origin             TEXT NOT NULL,
    language                   TEXT NOT NULL,
    narration_language         TEXT NOT NULL,
    policy_version             INTEGER NOT NULL DEFAULT 1,
    created_by                 UUID REFERENCES users (id),
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE story_workflow_settings (
    story_id               UUID PRIMARY KEY REFERENCES stories (id) ON DELETE CASCADE,
    batch_generation_size  INTEGER,
    creative_autonomy      TEXT,
    preferred_text_provider TEXT,
    preferred_text_model   TEXT,
    preferred_tts_provider TEXT,
    preferred_voice_id     TEXT,
    pause_before_tts       BOOLEAN NOT NULL DEFAULT FALSE,
    auto_ai_review         BOOLEAN NOT NULL DEFAULT TRUE,
    planning_horizon       INTEGER,
    fallback_policy        JSONB,
    updated_by             UUID REFERENCES users (id),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- Seed baseline roles
-- ============================================================

INSERT INTO roles (id, code) VALUES
    (gen_random_uuid(), 'USER'),
    (gen_random_uuid(), 'ADMIN');
