-- name: CreateStory :one
INSERT INTO stories (id, slug, title, description, status, visibility, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, slug, title, description, status, visibility, planning_mode, planning_phase,
          public_rating, public_warnings, cover_asset_id, current_story_bible_version_id,
          current_ending_plan_version_id, current_content_profile_version_id,
          current_official_canon_version_id, public_since, last_published_at,
          status_before_archive, created_by, created_at, updated_at, archived_at;

-- name: SlugExists :one
SELECT EXISTS(SELECT 1 FROM stories WHERE slug = $1);

-- name: GetStory :one
SELECT id, slug, title, description, status, visibility, planning_mode, planning_phase,
       public_rating, public_warnings, cover_asset_id, current_story_bible_version_id,
       current_ending_plan_version_id, current_content_profile_version_id,
       current_official_canon_version_id, public_since, last_published_at,
       status_before_archive, created_by, created_at, updated_at, archived_at
FROM stories
WHERE id = $1;

-- name: UpdateStory :one
UPDATE stories
SET title = $2, description = $3, status = $4, visibility = $5,
    status_before_archive = $6, cover_asset_id = $7, updated_at = NOW()
WHERE id = $1
RETURNING id, slug, title, description, status, visibility, planning_mode, planning_phase,
          public_rating, public_warnings, cover_asset_id, current_story_bible_version_id,
          current_ending_plan_version_id, current_content_profile_version_id,
          current_official_canon_version_id, public_since, last_published_at,
          status_before_archive, created_by, created_at, updated_at, archived_at;

-- name: ListStories :many
SELECT id, slug, title, description, status, visibility, planning_mode, planning_phase,
       public_rating, public_warnings, cover_asset_id, current_story_bible_version_id,
       current_ending_plan_version_id, current_content_profile_version_id,
       current_official_canon_version_id, public_since, last_published_at,
       status_before_archive, created_by, created_at, updated_at, archived_at
FROM stories
WHERE ($1::boolean = FALSE OR visibility = 'PUBLIC')
ORDER BY created_at DESC;

-- name: ListGenres :many
SELECT id, slug, name FROM genres ORDER BY name;

-- name: CreateGenerationPolicy :exec
INSERT INTO story_generation_policies (story_id, minimum_audio_duration_sec, target_audio_duration_sec,
                                       content_origin, language, narration_language, policy_version, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: HasGenerationPolicy :one
SELECT EXISTS(SELECT 1 FROM story_generation_policies WHERE story_id = $1);

-- name: GetWorkflowSettings :one
SELECT story_id, batch_generation_size, creative_autonomy, preferred_text_provider,
       preferred_text_model, preferred_tts_provider, preferred_voice_id, pause_before_tts,
       auto_ai_review, planning_horizon, fallback_policy, updated_by, updated_at
FROM story_workflow_settings
WHERE story_id = $1;

-- name: UpsertWorkflowSettings :one
INSERT INTO story_workflow_settings (story_id, batch_generation_size, creative_autonomy,
                                     preferred_text_provider, preferred_text_model,
                                     preferred_tts_provider, preferred_voice_id, pause_before_tts,
                                     auto_ai_review, planning_horizon, fallback_policy, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (story_id) DO UPDATE
SET batch_generation_size = EXCLUDED.batch_generation_size,
    creative_autonomy = EXCLUDED.creative_autonomy,
    preferred_text_provider = EXCLUDED.preferred_text_provider,
    preferred_text_model = EXCLUDED.preferred_text_model,
    preferred_tts_provider = EXCLUDED.preferred_tts_provider,
    preferred_voice_id = EXCLUDED.preferred_voice_id,
    pause_before_tts = EXCLUDED.pause_before_tts,
    auto_ai_review = EXCLUDED.auto_ai_review,
    planning_horizon = EXCLUDED.planning_horizon,
    fallback_policy = EXCLUDED.fallback_policy,
    updated_by = EXCLUDED.updated_by,
    updated_at = NOW()
RETURNING story_id, batch_generation_size, creative_autonomy, preferred_text_provider,
          preferred_text_model, preferred_tts_provider, preferred_voice_id, pause_before_tts,
          auto_ai_review, planning_horizon, fallback_policy, updated_by, updated_at;

-- name: NextContentProfileVersion :one
SELECT COALESCE(MAX(version_no), 0) + 1
FROM story_content_profile_versions
WHERE story_id = $1;

-- name: CreateContentProfileVersion :one
INSERT INTO story_content_profile_versions (id, story_id, version_no, profile, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, story_id, version_no, profile, base_policy_version_id, created_by, created_at;

-- name: GetCurrentContentProfile :one
SELECT id, story_id, version_no, profile, base_policy_version_id, created_by, created_at
FROM story_content_profile_versions
WHERE story_id = $1
ORDER BY version_no DESC
LIMIT 1;

-- name: CreateStoryAsset :one
INSERT INTO story_assets (id, story_id, type, storage_key, mime_type, size_bytes, status, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, story_id, type, storage_key, mime_type, size_bytes, checksum, rights_status, status, created_by, created_at;

-- name: LinkCoverAsset :exec
UPDATE stories SET cover_asset_id = $2, updated_at = NOW() WHERE id = $1;

-- name: SearchStories :many
SELECT id, slug, title, description, status, visibility, planning_mode, planning_phase,
       public_rating, public_warnings, cover_asset_id, current_story_bible_version_id,
       current_ending_plan_version_id, current_content_profile_version_id,
       current_official_canon_version_id, public_since, last_published_at,
       status_before_archive, created_by, created_at, updated_at, archived_at
FROM stories
WHERE visibility = 'PUBLIC'
  AND ($1::text = '' OR title ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%')
  AND ($2::text = '' OR id IN (
        SELECT sg.story_id FROM story_genres sg
        JOIN genres g ON g.id = sg.genre_id
        WHERE g.slug = $2
      ))
ORDER BY
  CASE WHEN $3::text = 'TITLE' THEN title END ASC,
  CASE WHEN $3::text = 'NEW' THEN public_since END DESC NULLS LAST,
  CASE WHEN $3::text = 'RECENTLY_UPDATED' THEN last_published_at END DESC NULLS LAST,
  created_at DESC;
