-- Durable chapter-scoped media version allocation.
--
-- A reservation is intentionally never rolled back/reused after it is returned
-- to application code. Gaps are acceptable: uniqueness and immutable object-key
-- ownership are more important than contiguous numbering.
CREATE TABLE chapter_media_version_counters (
    chapter_id                UUID PRIMARY KEY REFERENCES chapters (id) ON DELETE CASCADE,
    next_narration_revision   INTEGER NOT NULL CHECK (next_narration_revision > 0),
    next_audio_version        INTEGER NOT NULL CHECK (next_audio_version > 0)
);
