-- name: CreateNarrationRevision :one
INSERT INTO narration_revisions (
    id, chapter_id, content_revision_id, voice_profile_id, status,
    script_text, provider, model, generation_run_id
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9
)
RETURNING id, chapter_id, content_revision_id, voice_profile_id, status,
          script_text, provider, model, generation_run_id, created_at;

-- name: GetNarrationRevision :one
SELECT id, chapter_id, content_revision_id, voice_profile_id, status,
       script_text, provider, model, generation_run_id, created_at
FROM narration_revisions
WHERE id = $1;

-- name: ListNarrationRevisionsByChapter :many
SELECT id, chapter_id, content_revision_id, voice_profile_id, status,
       script_text, provider, model, generation_run_id, created_at
FROM narration_revisions
WHERE chapter_id = $1
ORDER BY created_at DESC, id DESC;

-- name: CreateAudioAsset :one
INSERT INTO audio_assets (
    id, chapter_id, version_no, source_narration_revision_id, status,
    storage_key, mime_type, size_bytes, duration_ms, bitrate_kbps,
    checksum, is_active, generation_run_id
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10,
    $11, $12, $13
)
RETURNING id, chapter_id, version_no, source_narration_revision_id, status, storage_key,
          mime_type, size_bytes, duration_ms, bitrate_kbps, checksum, is_active,
          generation_run_id, created_at;

-- name: CreateAudioAssetWithAllocatedVersion :one
WITH chapter_lock AS (
    SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))
), next_version AS (
    SELECT COALESCE(MAX(version_no), 0) + 1 AS version_no
    FROM audio_assets
    WHERE chapter_id = $1
)
INSERT INTO audio_assets (
    id, chapter_id, version_no, source_narration_revision_id, status,
    storage_key, mime_type, size_bytes, duration_ms, bitrate_kbps,
    checksum, is_active, generation_run_id
)
SELECT
    $2, $1, next_version.version_no, $3, $4,
    $5, $6, $7, $8, $9,
    $10, false, $11
FROM chapter_lock, next_version
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
