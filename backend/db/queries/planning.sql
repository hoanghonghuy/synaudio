-- ============================================================
-- Story Bible
-- ============================================================

-- name: NextBibleVersion :one
SELECT COALESCE(MAX(version_no), 0) + 1
FROM story_bible_versions
WHERE story_id = $1;

-- name: CreateBibleVersion :one
INSERT INTO story_bible_versions (id, story_id, version_no, content, based_on_version_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, story_id, version_no, content, based_on_version_id, created_by, generation_run_id, created_at;

-- name: GetCurrentBible :one
SELECT id, story_id, version_no, content, based_on_version_id, created_by, generation_run_id, created_at
FROM story_bible_versions
WHERE story_id = $1
ORDER BY version_no DESC
LIMIT 1;

-- ============================================================
-- Ending Plan
-- ============================================================

-- name: NextEndingVersion :one
SELECT COALESCE(MAX(version_no), 0) + 1
FROM story_ending_plan_versions
WHERE story_id = $1;

-- name: CreateEndingVersion :one
INSERT INTO story_ending_plan_versions (id, story_id, version_no, content, based_on_version_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, story_id, version_no, content, based_on_version_id, created_by, generation_run_id, created_at;

-- name: GetCurrentEnding :one
SELECT id, story_id, version_no, content, based_on_version_id, created_by, generation_run_id, created_at
FROM story_ending_plan_versions
WHERE story_id = $1
ORDER BY version_no DESC
LIMIT 1;

-- ============================================================
-- Story Arcs
-- ============================================================

-- name: NextArcOrdinal :one
SELECT COALESCE(MAX(ordinal), 0) + 1
FROM story_arcs
WHERE story_id = $1;

-- name: CreateArc :one
INSERT INTO story_arcs (id, story_id, ordinal, status)
VALUES ($1, $2, $3, $4)
RETURNING id, story_id, ordinal, status, current_version_id, created_at;

-- name: NextArcVersion :one
SELECT COALESCE(MAX(version_no), 0) + 1
FROM story_arc_versions
WHERE arc_id = $1;

-- name: CreateArcVersion :one
INSERT INTO story_arc_versions (id, arc_id, version_no, content, base_canon_version_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, arc_id, version_no, content, base_canon_version_id, generation_run_id, created_by, created_at;

-- name: GetArc :one
SELECT id, story_id, ordinal, status, current_version_id, created_at
FROM story_arcs
WHERE id = $1;

-- name: ListArcs :many
SELECT id, story_id, ordinal, status, current_version_id, created_at
FROM story_arcs
WHERE story_id = $1
ORDER BY ordinal;

-- ============================================================
-- Characters
-- ============================================================

-- name: CreateCharacter :one
INSERT INTO characters (id, story_id, canonical_name, importance)
VALUES ($1, $2, $3, $4)
RETURNING id, story_id, canonical_name, importance, current_profile_version_id, created_at;

-- name: NextProfileVersion :one
SELECT COALESCE(MAX(version_no), 0) + 1
FROM character_profile_versions
WHERE character_id = $1;

-- name: CreateProfileVersion :one
INSERT INTO character_profile_versions (id, character_id, version_no, profile, base_canon_version_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, character_id, version_no, profile, base_canon_version_id, created_by, generation_run_id, created_at;

-- name: ListCharacters :many
SELECT id, story_id, canonical_name, importance, current_profile_version_id, created_at
FROM characters
WHERE story_id = $1
ORDER BY created_at;

-- name: GetCharacter :one
SELECT id, story_id, canonical_name, importance, current_profile_version_id, created_at
FROM characters
WHERE id = $1;
