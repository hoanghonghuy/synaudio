package pgstore

import (
	"context"

	"github.com/synaudio/synaudio/backend/internal/listener"
)

// ListLibraryProgress returns the user's server-backed listening history ordered
// by the authoritative last-listened timestamp. Only currently public/published
// material is projected to the listener surface.
func (s *ListenerStore) ListLibraryProgress(ctx context.Context, userID string) ([]listener.LibraryItem, error) {
	rows, err := s.q.DBTX().Query(ctx, `
SELECT
    s.id::text,
    s.slug,
    s.title,
    COALESCE(s.description, ''),
    c.id::text,
    c.chapter_number,
    COALESCE(c.title, ''),
    p.position_ms,
    COALESCE(p.completed_at::text, ''),
    COALESCE(p.relisten_status, 'NO_RELISTEN_NEEDED'),
    COALESCE(p.last_listened_at::text, ''),
    COALESCE(p.updated_at::text, ''),
    COALESCE(aa.id::text, ''),
    COALESCE(aa.duration_ms, 0)
FROM listening_progress p
JOIN chapters c ON c.id = p.chapter_id
JOIN stories s ON s.id = c.story_id
LEFT JOIN audio_assets aa
       ON aa.id = c.current_audio_asset_id
      AND aa.status = 'READY'
WHERE p.user_id = $1
  AND s.visibility = 'PUBLIC'
  AND s.status IN ('ACTIVE', 'COMPLETED')
  AND c.published_at IS NOT NULL
ORDER BY p.last_listened_at DESC NULLS LAST, p.updated_at DESC, c.chapter_number DESC
`, toUUID(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]listener.LibraryItem, 0)
	for rows.Next() {
		var item listener.LibraryItem
		if err := rows.Scan(
			&item.StoryID,
			&item.StorySlug,
			&item.StoryTitle,
			&item.StoryDescription,
			&item.ChapterID,
			&item.ChapterNumber,
			&item.ChapterTitle,
			&item.PositionMs,
			&item.CompletedAt,
			&item.RelistenStatus,
			&item.LastListenedAt,
			&item.UpdatedAt,
			&item.AudioAssetID,
			&item.AudioDurationMs,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *ListenerStore) ListFavoriteStories(ctx context.Context, userID string) ([]listener.FavoriteStory, error) {
	rows, err := s.q.DBTX().Query(ctx, `
SELECT
    s.id::text,
    s.slug,
    s.title,
    COALESCE(s.description, ''),
    f.created_at::text
FROM favorites f
JOIN stories s ON s.id = f.story_id
WHERE f.user_id = $1
  AND s.visibility = 'PUBLIC'
  AND s.status IN ('ACTIVE', 'COMPLETED')
ORDER BY f.created_at DESC, s.id
`, toUUID(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]listener.FavoriteStory, 0)
	for rows.Next() {
		var item listener.FavoriteStory
		if err := rows.Scan(&item.StoryID, &item.Slug, &item.Title, &item.Description, &item.FavoritedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
