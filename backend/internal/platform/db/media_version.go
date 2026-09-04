package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const reserveNarrationRevision = `
INSERT INTO chapter_media_version_counters (
    chapter_id, next_narration_revision, next_audio_version
)
VALUES (
    $1,
    (SELECT COALESCE(MAX(revision_no), 0) + 2 FROM narration_revisions WHERE chapter_id = $1),
    (SELECT COALESCE(MAX(version_no), 0) + 1 FROM audio_assets WHERE chapter_id = $1)
)
ON CONFLICT (chapter_id) DO UPDATE
SET next_narration_revision = chapter_media_version_counters.next_narration_revision + 1
RETURNING next_narration_revision - 1`

const reserveAudioVersion = `
INSERT INTO chapter_media_version_counters (
    chapter_id, next_narration_revision, next_audio_version
)
VALUES (
    $1,
    (SELECT COALESCE(MAX(revision_no), 0) + 1 FROM narration_revisions WHERE chapter_id = $1),
    (SELECT COALESCE(MAX(version_no), 0) + 2 FROM audio_assets WHERE chapter_id = $1)
)
ON CONFLICT (chapter_id) DO UPDATE
SET next_audio_version = chapter_media_version_counters.next_audio_version + 1
RETURNING next_audio_version - 1`

// ReserveNarrationRevision durably advances the chapter-scoped narration
// counter. A returned number is intentionally never reused, even if the caller
// later fails, so concurrent attempts cannot receive the same immutable identity.
func ReserveNarrationRevision(ctx context.Context, executor DBTX, chapterID pgtype.UUID) (int32, error) {
	var revision int32
	err := executor.QueryRow(ctx, reserveNarrationRevision, chapterID).Scan(&revision)
	return revision, err
}

// ReserveAudioVersion durably advances the chapter-scoped audio counter. The
// reservation is committed independently of later object/metadata work so a
// failed attempt cannot expose the same version/object-key identity to a retry.
func ReserveAudioVersion(ctx context.Context, executor DBTX, chapterID pgtype.UUID) (int32, error) {
	var version int32
	err := executor.QueryRow(ctx, reserveAudioVersion, chapterID).Scan(&version)
	return version, err
}
