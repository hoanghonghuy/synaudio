-- ============================================================
-- Favorites
-- ============================================================

-- name: AddFavorite :exec
INSERT INTO favorites (user_id, story_id)
VALUES ($1, $2)
ON CONFLICT (user_id, story_id) DO NOTHING;

-- name: RemoveFavorite :exec
DELETE FROM favorites
WHERE user_id = $1 AND story_id = $2;

-- name: IsFavorite :one
SELECT EXISTS(SELECT 1 FROM favorites WHERE user_id = $1 AND story_id = $2);

-- name: ListFavorites :many
SELECT user_id, story_id, created_at
FROM favorites
WHERE user_id = $1
ORDER BY created_at DESC;

-- ============================================================
-- Listening Progress
-- ============================================================

-- name: GetProgress :one
SELECT user_id, chapter_id, position_ms, completed_at, last_audio_asset_id,
       last_playback_session_id, version, relisten_status, last_listened_at, updated_at
FROM listening_progress
WHERE user_id = $1 AND chapter_id = $2;

-- name: SaveProgress :one
INSERT INTO listening_progress (user_id, chapter_id, position_ms, last_audio_asset_id,
                                last_playback_session_id, version, last_listened_at)
SELECT $1, $2, $3, $4, $5, sqlc.arg('expected_version') + 1, NOW()
WHERE sqlc.arg('expected_version') = 0
   OR EXISTS (SELECT 1 FROM listening_progress lp WHERE lp.user_id = $1 AND lp.chapter_id = $2)
ON CONFLICT (user_id, chapter_id)
DO UPDATE SET position_ms = EXCLUDED.position_ms,
              last_audio_asset_id = EXCLUDED.last_audio_asset_id,
              last_playback_session_id = EXCLUDED.last_playback_session_id,
              version = EXCLUDED.version,
              last_listened_at = NOW(),
              updated_at = NOW()
WHERE listening_progress.version = sqlc.arg('expected_version')
RETURNING user_id, chapter_id, position_ms, completed_at, last_audio_asset_id,
          last_playback_session_id, version, relisten_status, last_listened_at, updated_at;

-- name: MarkCompleted :one
UPDATE listening_progress
SET completed_at = NOW(),
    updated_at = NOW()
WHERE user_id = $1 AND chapter_id = $2
RETURNING user_id, chapter_id, position_ms, completed_at, last_audio_asset_id,
          last_playback_session_id, version, relisten_status, last_listened_at, updated_at;

-- name: ApplyRelistenStatus :many
UPDATE listening_progress
SET relisten_status = $2,
    updated_at = NOW()
WHERE chapter_id = $1
RETURNING user_id, chapter_id, position_ms, completed_at, last_audio_asset_id,
          last_playback_session_id, version, relisten_status, last_listened_at, updated_at;
