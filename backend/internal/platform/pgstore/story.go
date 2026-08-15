package pgstore

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/platform/db"
	"github.com/synaudio/synaudio/backend/internal/story"
)

// StoryStore implements story.Store backed by PostgreSQL via sqlc.
type StoryStore struct {
	q *db.Queries
}

func NewStoryStore(q *db.Queries) *StoryStore {
	return &StoryStore{q: q}
}

func (s *StoryStore) CreateStory(ctx context.Context, st story.Story) (story.Story, error) {
	row, err := s.q.CreateStory(ctx, db.CreateStoryParams{
		ID:          toUUID(st.ID),
		Slug:        st.Slug,
		Title:       st.Title,
		Description: toText(st.Description),
		Status:      st.Status,
		Visibility:  st.Visibility,
		CreatedBy:   toUUID(st.CreatedBy),
	})
	if err != nil {
		return story.Story{}, err
	}
	return toStory(row), nil
}

func (s *StoryStore) CreateGenerationPolicy(ctx context.Context, p story.GenerationPolicy) error {
	return s.q.CreateGenerationPolicy(ctx, db.CreateGenerationPolicyParams{
		StoryID:                 toUUID(p.StoryID),
		MinimumAudioDurationSec: int32(p.MinimumAudioDurationSec),
		TargetAudioDurationSec:  int32(p.TargetAudioDurationSec),
		ContentOrigin:           p.ContentOrigin,
		Language:                p.Language,
		NarrationLanguage:       p.NarrationLanguage,
		PolicyVersion:           int32(p.PolicyVersion),
		CreatedBy:               toUUID(p.CreatedBy),
	})
}

func (s *StoryStore) HasGenerationPolicy(ctx context.Context, storyID string) (bool, error) {
	return s.q.HasGenerationPolicy(ctx, toUUID(storyID))
}

func (s *StoryStore) SlugExists(ctx context.Context, slug string) (bool, error) {
	return s.q.SlugExists(ctx, slug)
}

func (s *StoryStore) ListGenres(ctx context.Context) ([]story.Genre, error) {
	rows, err := s.q.ListGenres(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]story.Genre, 0, len(rows))
	for _, g := range rows {
		out = append(out, story.Genre{
			ID:   fromUUID(g.ID),
			Slug: g.Slug,
			Name: g.Name,
		})
	}
	return out, nil
}

func (s *StoryStore) ListStories(ctx context.Context, publicOnly bool) ([]story.Story, error) {
	rows, err := s.q.ListStories(ctx, publicOnly)
	if err != nil {
		return nil, err
	}
	out := make([]story.Story, 0, len(rows))
	for _, r := range rows {
		out = append(out, toStory(r))
	}
	return out, nil
}

func (s *StoryStore) GetStory(ctx context.Context, storyID string) (story.Story, error) {
	row, err := s.q.GetStory(ctx, toUUID(storyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return story.Story{}, story.ErrStoryNotFound
		}
		return story.Story{}, err
	}
	return toStory(row), nil
}

func (s *StoryStore) UpdateStory(ctx context.Context, st story.Story) (story.Story, error) {
	row, err := s.q.UpdateStory(ctx, db.UpdateStoryParams{
		ID:                  toUUID(st.ID),
		Title:               st.Title,
		Description:         toText(st.Description),
		Status:              st.Status,
		Visibility:          st.Visibility,
		StatusBeforeArchive: toText(st.StatusBeforeArchive),
		CoverAssetID:        toUUID(st.CoverAssetID),
	})
	if err != nil {
		return story.Story{}, err
	}
	return toStory(row), nil
}

func (s *StoryStore) GetWorkflowSettings(ctx context.Context, storyID string) (story.WorkflowSettings, error) {
	row, err := s.q.GetWorkflowSettings(ctx, toUUID(storyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return story.WorkflowSettings{}, story.ErrStoryNotFound
		}
		return story.WorkflowSettings{}, err
	}
	return toWorkflowSettings(row), nil
}

func (s *StoryStore) UpdateWorkflowSettings(ctx context.Context, ws story.WorkflowSettings) (story.WorkflowSettings, error) {
	fallback, _ := json.Marshal(ws.FallbackPolicy)
	row, err := s.q.UpsertWorkflowSettings(ctx, db.UpsertWorkflowSettingsParams{
		StoryID:               toUUID(ws.StoryID),
		BatchGenerationSize:   toInt4(ws.BatchGenerationSize),
		CreativeAutonomy:      toText(ws.CreativeAutonomy),
		PreferredTextProvider: toText(ws.PreferredTextProvider),
		PreferredTextModel:    toText(ws.PreferredTextModel),
		PreferredTtsProvider:  toText(ws.PreferredTTSProvider),
		PreferredVoiceID:      toText(ws.PreferredVoiceID),
		PauseBeforeTts:        ws.PauseBeforeTTS,
		AutoAiReview:          ws.AutoAIReview,
		PlanningHorizon:       toInt4(ws.PlanningHorizon),
		FallbackPolicy:        fallback,
		UpdatedBy:             toUUID(ws.UpdatedBy),
	})
	if err != nil {
		return story.WorkflowSettings{}, err
	}
	return toWorkflowSettings(row), nil
}

func (s *StoryStore) NextContentProfileVersion(ctx context.Context, storyID string) (int, error) {
	n, err := s.q.NextContentProfileVersion(ctx, toUUID(storyID))
	return int(n), err
}

func (s *StoryStore) CreateContentProfileVersion(ctx context.Context, cp story.ContentProfileVersion) (story.ContentProfileVersion, error) {
	profile, _ := json.Marshal(cp.Profile)
	row, err := s.q.CreateContentProfileVersion(ctx, db.CreateContentProfileVersionParams{
		ID:        toUUID(cp.ID),
		StoryID:   toUUID(cp.StoryID),
		VersionNo: int32(cp.VersionNo),
		Profile:   profile,
		CreatedBy: toUUID(cp.CreatedBy),
	})
	if err != nil {
		return story.ContentProfileVersion{}, err
	}
	return toContentProfile(row), nil
}

func (s *StoryStore) GetCurrentContentProfile(ctx context.Context, storyID string) (story.ContentProfileVersion, error) {
	row, err := s.q.GetCurrentContentProfile(ctx, toUUID(storyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return story.ContentProfileVersion{}, story.ErrContentProfileNotFound
		}
		return story.ContentProfileVersion{}, err
	}
	return toContentProfile(row), nil
}

func (s *StoryStore) CreateStoryAsset(ctx context.Context, a story.StoryAsset) (story.StoryAsset, error) {
	row, err := s.q.CreateStoryAsset(ctx, db.CreateStoryAssetParams{
		ID:         toUUID(a.ID),
		StoryID:    toUUID(a.StoryID),
		Type:       a.Type,
		StorageKey: a.StorageKey,
		MimeType:   toText(a.MimeType),
		SizeBytes:  pgtypeInt8(a.SizeBytes),
		Status:     a.Status,
		CreatedBy:  toUUID(a.CreatedBy),
	})
	if err != nil {
		return story.StoryAsset{}, err
	}
	return toStoryAsset(row), nil
}

func (s *StoryStore) LinkCoverAsset(ctx context.Context, storyID, assetID string) error {
	return s.q.LinkCoverAsset(ctx, db.LinkCoverAssetParams{
		ID:           toUUID(storyID),
		CoverAssetID: toUUID(assetID),
	})
}

func (s *StoryStore) SearchStories(ctx context.Context, in story.SearchStoriesInput) ([]story.Story, error) {
	rows, err := s.q.SearchStories(ctx, db.SearchStoriesParams{
		Column1: in.Query,
		Column2: in.Genre,
		Column3: in.Sort,
	})
	if err != nil {
		return nil, err
	}
	out := make([]story.Story, 0, len(rows))
	for _, r := range rows {
		out = append(out, toStory(r))
	}
	return out, nil
}

func toStory(row db.Story) story.Story {
	return story.Story{
		ID:                 fromUUID(row.ID),
		Slug:               row.Slug,
		Title:              row.Title,
		Description:        fromText(row.Description),
		Status:             row.Status,
		Visibility:         row.Visibility,
		PlanningMode:       row.PlanningMode,
		StatusBeforeArchive: fromText(row.StatusBeforeArchive),
		CoverAssetID:       fromUUID(row.CoverAssetID),
		CreatedBy:          fromUUID(row.CreatedBy),
	}
}

func toWorkflowSettings(row db.StoryWorkflowSetting) story.WorkflowSettings {
	var fallback map[string]any
	_ = json.Unmarshal(row.FallbackPolicy, &fallback)
	return story.WorkflowSettings{
		StoryID:              fromUUID(row.StoryID),
		BatchGenerationSize:  int(row.BatchGenerationSize.Int32),
		CreativeAutonomy:     fromText(row.CreativeAutonomy),
		PreferredTextProvider: fromText(row.PreferredTextProvider),
		PreferredTextModel:    fromText(row.PreferredTextModel),
		PreferredTTSProvider:  fromText(row.PreferredTtsProvider),
		PreferredVoiceID:      fromText(row.PreferredVoiceID),
		PauseBeforeTTS:        row.PauseBeforeTts,
		AutoAIReview:          row.AutoAiReview,
		PlanningHorizon:       int(row.PlanningHorizon.Int32),
		FallbackPolicy:        fallback,
		UpdatedBy:             fromUUID(row.UpdatedBy),
	}
}

func toContentProfile(row db.StoryContentProfileVersion) story.ContentProfileVersion {
	var profile map[string]any
	_ = json.Unmarshal(row.Profile, &profile)
	return story.ContentProfileVersion{
		ID:        fromUUID(row.ID),
		StoryID:   fromUUID(row.StoryID),
		VersionNo: int(row.VersionNo),
		Profile:   profile,
		CreatedBy: fromUUID(row.CreatedBy),
	}
}

func toStoryAsset(row db.StoryAsset) story.StoryAsset {
	return story.StoryAsset{
		ID:          fromUUID(row.ID),
		StoryID:     fromUUID(row.StoryID),
		Type:        row.Type,
		StorageKey:  row.StorageKey,
		MimeType:    fromText(row.MimeType),
		SizeBytes:   row.SizeBytes.Int64,
		Status:      row.Status,
		CreatedBy:   fromUUID(row.CreatedBy),
	}
}
