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

-- ============================================================
-- Chapters
-- ============================================================

-- name: NextChapterNumber :one
SELECT COALESCE(MAX(chapter_number), 0) + 1
FROM chapters
WHERE story_id = $1;

-- name: CreateChapter :one
INSERT INTO chapters (id, story_id, chapter_number, title, status, arc_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, story_id, chapter_number, title, status, arc_id, current_plan_revision_id,
          current_content_revision_id, current_narration_revision_id, current_audio_asset_id,
          official_canon_version_id, published_at, archived_at, created_at, updated_at;

-- name: NextPlanRevision :one
SELECT COALESCE(MAX(revision_no), 0) + 1
FROM chapter_plan_revisions
WHERE chapter_id = $1;

-- name: CreatePlanRevision :one
INSERT INTO chapter_plan_revisions (id, chapter_id, revision_no, plan, base_canon_version_id, arc_version_id, source_type, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, chapter_id, revision_no, plan, base_canon_version_id, arc_version_id, source_type, generation_run_id, created_by, created_at;

-- name: GetChapter :one
SELECT id, story_id, chapter_number, title, status, arc_id, current_plan_revision_id,
       current_content_revision_id, current_narration_revision_id, current_audio_asset_id,
       official_canon_version_id, published_at, archived_at, created_at, updated_at
FROM chapters
WHERE id = $1;

-- name: ListChapters :many
SELECT id, story_id, chapter_number, title, status, arc_id, current_plan_revision_id,
       current_content_revision_id, current_narration_revision_id, current_audio_asset_id,
       official_canon_version_id, published_at, archived_at, created_at, updated_at
FROM chapters
WHERE story_id = $1
ORDER BY chapter_number;

-- name: UpdateChapterStatus :one
UPDATE chapters
SET status = $2,
    published_at = CASE WHEN $2 = 'PUBLISHED' THEN NOW() ELSE published_at END,
    updated_at = NOW()
WHERE id = $1
RETURNING id, story_id, chapter_number, title, status, arc_id, current_plan_revision_id,
          current_content_revision_id, current_narration_revision_id, current_audio_asset_id,
          official_canon_version_id, published_at, archived_at, created_at, updated_at;

-- ============================================================
-- StoryFacts
-- ============================================================

-- name: CreateFact :one
INSERT INTO story_facts (id, story_id, subject_type, subject_id, fact_type, value, importance, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, story_id, subject_type, subject_id, fact_type, value, importance, status,
          valid_from_canon_version_id, invalidated_at_canon_version_id, supersedes_fact_id,
          source_chapter_id, source_content_revision_id, generation_run_id, created_at;

-- name: ListFacts :many
SELECT id, story_id, subject_type, subject_id, fact_type, value, importance, status,
       valid_from_canon_version_id, invalidated_at_canon_version_id, supersedes_fact_id,
       source_chapter_id, source_content_revision_id, generation_run_id, created_at
FROM story_facts
WHERE story_id = $1
ORDER BY created_at;

-- ============================================================
-- PlotThreads
-- ============================================================

-- name: CreatePlotThread :one
INSERT INTO plot_threads (id, story_id, title, summary, importance, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, story_id, title, summary, importance, status,
          opened_chapter_id, resolved_chapter_id, last_advanced_chapter_id, created_at, updated_at;

-- name: ListPlotThreads :many
SELECT id, story_id, title, summary, importance, status,
       opened_chapter_id, resolved_chapter_id, last_advanced_chapter_id, created_at, updated_at
FROM plot_threads
WHERE story_id = $1
ORDER BY created_at;

-- name: CreatePlotThreadEvent :one
INSERT INTO plot_thread_events (id, plot_thread_id, canon_version_id, chapter_id, event_type, detail)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, plot_thread_id, canon_version_id, chapter_id, event_type, detail, created_at;

-- name: ListPlotThreadEvents :many
SELECT id, plot_thread_id, canon_version_id, chapter_id, event_type, detail, created_at
FROM plot_thread_events
WHERE plot_thread_id = $1
ORDER BY created_at;

-- ============================================================
-- Canon branches + versions
-- ============================================================

-- name: CreateCanonBranch :one
INSERT INTO canon_branches (id, story_id, type, status)
VALUES ($1, $2, $3, $4)
RETURNING id, story_id, type, status, base_version_id, generation_run_id, retcon_request_id, created_at;

-- name: NextCanonSequence :one
SELECT COALESCE(MAX(sequence_no), 0) + 1
FROM canon_versions
WHERE branch_id = $1;

-- name: CreateCanonVersion :one
INSERT INTO canon_versions (id, story_id, branch_id, sequence_no, parent_version_id, source_chapter_id, source_content_revision_id, source_provisional_version_id, status, committed_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, story_id, branch_id, sequence_no, parent_version_id, source_chapter_id,
          source_content_revision_id, source_provisional_version_id, generation_run_id,
          retcon_request_id, status, committed_by, created_at;

-- name: ListCanonVersions :many
SELECT id, story_id, branch_id, sequence_no, parent_version_id, source_chapter_id,
       source_content_revision_id, source_provisional_version_id, generation_run_id,
       retcon_request_id, status, committed_by, created_at
FROM canon_versions
WHERE branch_id = $1
ORDER BY sequence_no;

-- name: GetCanonVersion :one
SELECT id, story_id, branch_id, sequence_no, parent_version_id, source_chapter_id,
       source_content_revision_id, source_provisional_version_id, generation_run_id,
       retcon_request_id, status, committed_by, created_at
FROM canon_versions
WHERE id = $1;

-- name: UpdateCanonVersion :one
UPDATE canon_versions
SET status = $2,
    committed_by = $3
WHERE id = $1
RETURNING id, story_id, branch_id, sequence_no, parent_version_id, source_chapter_id,
          source_content_revision_id, source_provisional_version_id, generation_run_id,
          retcon_request_id, status, committed_by, created_at;

-- name: CreateCanonChangeItem :one
INSERT INTO canon_change_items (id, canon_version_id, entity_type, entity_id, change_type, metadata)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, canon_version_id, entity_type, entity_id, change_type, metadata;

-- name: ListCanonChangeItems :many
SELECT id, canon_version_id, entity_type, entity_id, change_type, metadata
FROM canon_change_items
WHERE canon_version_id = $1
ORDER BY entity_type;

-- ============================================================
-- ContextSnapshots
-- ============================================================

-- name: CreateContextSnapshot :one
INSERT INTO context_snapshots (id, story_id, chapter_id, canon_version_id, bible_version_id,
                              ending_plan_version_id, arc_version_id, content_profile_version_id,
                              prompt_version, workflow_version, provider, model)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, run_id, story_id, chapter_id, canon_version_id, bible_version_id,
          ending_plan_version_id, arc_version_id, content_profile_version_id,
          prompt_version, workflow_version, provider, model, included_refs,
          historical_hits, admin_instruction, created_at;

-- name: ListContextSnapshots :many
SELECT id, run_id, story_id, chapter_id, canon_version_id, bible_version_id,
       ending_plan_version_id, arc_version_id, content_profile_version_id,
       prompt_version, workflow_version, provider, model, included_refs,
       historical_hits, admin_instruction, created_at
FROM context_snapshots
WHERE story_id = $1
ORDER BY created_at;

-- name: GetContextSnapshot :one
SELECT id, run_id, story_id, chapter_id, canon_version_id, bible_version_id,
       ending_plan_version_id, arc_version_id, content_profile_version_id,
       prompt_version, workflow_version, provider, model, included_refs,
       historical_hits, admin_instruction, created_at
FROM context_snapshots
WHERE id = $1;

-- ============================================================
-- Creative Decisions
-- ============================================================

-- name: CreateCreativeDecision :one
INSERT INTO creative_decisions (id, story_id, chapter_id, arc_id, origin, decision_type,
                                severity, status, blocking_level, question, context_summary,
                                created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, story_id, chapter_id, arc_id, origin, decision_type, severity, status,
          blocking_level, question, context_summary, recommended_option_id, selected_option_id,
          custom_selected_text, rejection_scope, revisit_condition, triggered_by_run_id,
          created_by, selected_by, created_at, selected_at, applied_at;

-- name: GetCreativeDecision :one
SELECT id, story_id, chapter_id, arc_id, origin, decision_type, severity, status,
       blocking_level, question, context_summary, recommended_option_id, selected_option_id,
       custom_selected_text, rejection_scope, revisit_condition, triggered_by_run_id,
       created_by, selected_by, created_at, selected_at, applied_at
FROM creative_decisions
WHERE id = $1;

-- name: ListCreativeDecisions :many
SELECT id, story_id, chapter_id, arc_id, origin, decision_type, severity, status,
       blocking_level, question, context_summary, recommended_option_id, selected_option_id,
       custom_selected_text, rejection_scope, revisit_condition, triggered_by_run_id,
       created_by, selected_by, created_at, selected_at, applied_at
FROM creative_decisions
WHERE story_id = $1
ORDER BY created_at;

-- name: UpdateCreativeDecision :one
UPDATE creative_decisions
SET status = $2,
    selected_option_id = $3,
    custom_selected_text = $4,
    rejection_scope = $5,
    selected_by = $6,
    selected_at = CASE WHEN $2 = 'SELECTED' THEN NOW() ELSE selected_at END,
    applied_at = CASE WHEN $2 = 'APPLIED' THEN NOW() ELSE applied_at END
WHERE id = $1
RETURNING id, story_id, chapter_id, arc_id, origin, decision_type, severity, status,
          blocking_level, question, context_summary, recommended_option_id, selected_option_id,
          custom_selected_text, rejection_scope, revisit_condition, triggered_by_run_id,
          created_by, selected_by, created_at, selected_at, applied_at;

-- ============================================================
-- Attention Items
-- ============================================================

-- name: CreateAttentionItem :one
INSERT INTO attention_items (id, story_id, chapter_id, priority, kind, title, detail, action)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, story_id, chapter_id, priority, kind, title, detail, action, resolved, created_at, resolved_at;

-- name: ListAttentionItems :many
SELECT id, story_id, chapter_id, priority, kind, title, detail, action, resolved, created_at, resolved_at
FROM attention_items
WHERE story_id = $1
ORDER BY created_at;

-- name: GetAttentionItem :one
SELECT id, story_id, chapter_id, priority, kind, title, detail, action, resolved, created_at, resolved_at
FROM attention_items
WHERE id = $1;

-- name: UpdateAttentionItem :one
UPDATE attention_items
SET resolved = $2,
    resolved_at = CASE WHEN $2 = TRUE THEN NOW() ELSE resolved_at END
WHERE id = $1
RETURNING id, story_id, chapter_id, priority, kind, title, detail, action, resolved, created_at, resolved_at;



