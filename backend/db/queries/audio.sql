-- ============================================================
-- Narration Revisions
-- ============================================================

-- name: NextNarrationRevision :one
SELECT COALESCE(MAX(revision_no), 0) + 1
FROM narration_revisions
WHERE chapter_id = $1;

-- name: CreateNarrationRevision :one
INSERT INTO narration_revisions (id, chapter_id, revision_no, source_content_revision_id,
                                 voice_id, script, status, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, chapter_id, revision_no, source_content_revision_id, voice_id, script,
          status, generation_run_id, created_by, created_at;

-- name: GetNarrationRevision :one
SELECT id, chapter_id, revision_no, source_content_revision_id, voice_id, script,
       status, generation_run_id, created_by, created_at
FROM narration_revisions
WHERE id = $1;

-- ============================================================
-- TTS Segments
-- ============================================================

-- name: CreateTTSSegment :one
INSERT INTO tts_segments (id, narration_revision_id, segment_no, text, direction,
                          status, provider, model, voice_id, duration_ms, temp_storage_key)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, narration_revision_id, segment_no, text, direction, status, provider,
          model, voice_id, duration_ms, temp_storage_key, generation_job_id, created_at;

-- name: GetTTSSegment :one
SELECT id, narration_revision_id, segment_no, text, direction, status, provider,
       model, voice_id, duration_ms, temp_storage_key, generation_job_id, created_at
FROM tts_segments
WHERE id = $1;

-- name: UpdateTTSSegment :one
UPDATE tts_segments
SET status = $2,
    provider = $3,
    model = $4,
    duration_ms = $5,
    temp_storage_key = $6
WHERE id = $1
RETURNING id, narration_revision_id, segment_no, text, direction, status, provider,
          model, voice_id, duration_ms, temp_storage_key, generation_job_id, created_at;

-- ============================================================
-- Audio Assets
-- ============================================================

-- name: NextAudioVersion :one
SELECT COALESCE(MAX(version_no), 0) + 1
FROM audio_assets
WHERE chapter_id = $1;

-- name: CreateAudioAsset :one
INSERT INTO audio_assets (id, chapter_id, version_no, source_narration_revision_id,
                          status, storage_key, mime_type, size_bytes, duration_ms,
                          bitrate_kbps, checksum, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, chapter_id, version_no, source_narration_revision_id, status, storage_key,
          mime_type, size_bytes, duration_ms, bitrate_kbps, checksum, is_active,
          generation_run_id, created_at;

-- name: GetAudioAsset :one
SELECT id, chapter_id, version_no, source_narration_revision_id, status, storage_key,
       mime_type, size_bytes, duration_ms, bitrate_kbps, checksum, is_active,
       generation_run_id, created_at
FROM audio_assets
WHERE id = $1;

-- name: GetActiveAudioAsset :one
SELECT id, chapter_id, version_no, source_narration_revision_id, status, storage_key,
       mime_type, size_bytes, duration_ms, bitrate_kbps, checksum, is_active,
       generation_run_id, created_at
FROM audio_assets
WHERE chapter_id = $1 AND is_active = true;

-- name: SetActiveAudioAsset :many
WITH eligible_target AS (
    SELECT aa.id
    FROM audio_assets AS aa
    WHERE aa.chapter_id = $1
      AND aa.id = $2
      AND aa.status = 'READY'
    FOR UPDATE
)
UPDATE audio_assets AS aa
SET is_active = (aa.id = $2)
WHERE aa.chapter_id = $1
  AND EXISTS (SELECT 1 FROM eligible_target)
RETURNING aa.id, aa.chapter_id, aa.version_no, aa.source_narration_revision_id, aa.status, aa.storage_key,
          aa.mime_type, aa.size_bytes, aa.duration_ms, aa.bitrate_kbps, aa.checksum, aa.is_active,
          aa.generation_run_id, aa.created_at;
