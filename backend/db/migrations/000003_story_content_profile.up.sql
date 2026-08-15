-- Story Content Profile versions (versioned + controlled)
-- Phase 1 scope: Story-specific content boundaries.

CREATE TABLE story_content_profile_versions (
    id                    UUID PRIMARY KEY,
    story_id              UUID NOT NULL REFERENCES stories (id) ON DELETE CASCADE,
    version_no            INTEGER NOT NULL,
    profile               JSONB NOT NULL,
    base_policy_version_id UUID,
    created_by            UUID REFERENCES users (id),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (story_id, version_no)
);

CREATE INDEX idx_story_content_profile_versions_story_id
    ON story_content_profile_versions (story_id);

-- Migration-safe circular pointer: stories.current_content_profile_version_id
ALTER TABLE stories
    ADD CONSTRAINT fk_stories_current_content_profile_version
    FOREIGN KEY (current_content_profile_version_id)
    REFERENCES story_content_profile_versions (id);
