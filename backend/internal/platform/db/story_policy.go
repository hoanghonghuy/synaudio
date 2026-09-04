package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// GenerationPolicyRow is the immutable Story generation policy projection used
// by admin planning/readiness surfaces. It intentionally exposes no mutation.
type GenerationPolicyRow struct {
	StoryID                 pgtype.UUID
	MinimumAudioDurationSec int32
	TargetAudioDurationSec  int32
	ContentOrigin           string
	Language                string
	NarrationLanguage       string
	PolicyVersion           int32
	CreatedBy               pgtype.UUID
}

func (q *Queries) GetGenerationPolicy(ctx context.Context, storyID pgtype.UUID) (GenerationPolicyRow, error) {
	row := q.db.QueryRow(ctx, `
SELECT story_id, minimum_audio_duration_sec, target_audio_duration_sec,
       content_origin, language, narration_language, policy_version, created_by
FROM story_generation_policies
WHERE story_id = $1`, storyID)

	var p GenerationPolicyRow
	err := row.Scan(
		&p.StoryID,
		&p.MinimumAudioDurationSec,
		&p.TargetAudioDurationSec,
		&p.ContentOrigin,
		&p.Language,
		&p.NarrationLanguage,
		&p.PolicyVersion,
		&p.CreatedBy,
	)
	return p, err
}
