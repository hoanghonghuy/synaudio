package pgstore

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/platform/db"
	"github.com/synaudio/synaudio/backend/internal/planning"
)

// PlanningStore implements planning.Store backed by PostgreSQL via sqlc.
type PlanningStore struct {
	q *db.Queries
}

func NewPlanningStore(q *db.Queries) *PlanningStore {
	return &PlanningStore{q: q}
}

// ============================================================
// Story Bible
// ============================================================

func (s *PlanningStore) NextBibleVersion(ctx context.Context, storyID string) (int, error) {
	n, err := s.q.NextBibleVersion(ctx, toUUID(storyID))
	return int(n), err
}

func (s *PlanningStore) CreateBibleVersion(ctx context.Context, v planning.StoryBibleVersion) (planning.StoryBibleVersion, error) {
	content, _ := json.Marshal(v.Content)
	row, err := s.q.CreateBibleVersion(ctx, db.CreateBibleVersionParams{
		ID:               toUUID(v.ID),
		StoryID:          toUUID(v.StoryID),
		VersionNo:        int32(v.VersionNo),
		Content:          content,
		BasedOnVersionID: toUUID(v.BasedOnVersionID),
		CreatedBy:        toUUID(v.CreatedBy),
	})
	if err != nil {
		return planning.StoryBibleVersion{}, err
	}
	return toBibleVersion(row), nil
}

func (s *PlanningStore) GetCurrentBible(ctx context.Context, storyID string) (planning.StoryBibleVersion, error) {
	row, err := s.q.GetCurrentBible(ctx, toUUID(storyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return planning.StoryBibleVersion{}, planning.ErrBibleNotFound
		}
		return planning.StoryBibleVersion{}, err
	}
	return toBibleVersion(row), nil
}

// ============================================================
// Ending Plan
// ============================================================

func (s *PlanningStore) NextEndingVersion(ctx context.Context, storyID string) (int, error) {
	n, err := s.q.NextEndingVersion(ctx, toUUID(storyID))
	return int(n), err
}

func (s *PlanningStore) CreateEndingVersion(ctx context.Context, v planning.EndingPlanVersion) (planning.EndingPlanVersion, error) {
	content, _ := json.Marshal(v.Content)
	row, err := s.q.CreateEndingVersion(ctx, db.CreateEndingVersionParams{
		ID:               toUUID(v.ID),
		StoryID:          toUUID(v.StoryID),
		VersionNo:        int32(v.VersionNo),
		Content:          content,
		BasedOnVersionID: toUUID(v.BasedOnVersionID),
		CreatedBy:        toUUID(v.CreatedBy),
	})
	if err != nil {
		return planning.EndingPlanVersion{}, err
	}
	return toEndingVersion(row), nil
}

func (s *PlanningStore) GetCurrentEnding(ctx context.Context, storyID string) (planning.EndingPlanVersion, error) {
	row, err := s.q.GetCurrentEnding(ctx, toUUID(storyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return planning.EndingPlanVersion{}, planning.ErrEndingNotFound
		}
		return planning.EndingPlanVersion{}, err
	}
	return toEndingVersion(row), nil
}

// ============================================================
// Story Arcs
// ============================================================

func (s *PlanningStore) NextArcOrdinal(ctx context.Context, storyID string) (int, error) {
	n, err := s.q.NextArcOrdinal(ctx, toUUID(storyID))
	return int(n), err
}

func (s *PlanningStore) CreateArc(ctx context.Context, a planning.StoryArc) (planning.StoryArc, error) {
	row, err := s.q.CreateArc(ctx, db.CreateArcParams{
		ID:      toUUID(a.ID),
		StoryID: toUUID(a.StoryID),
		Ordinal: int32(a.Ordinal),
		Status:  a.Status,
	})
	if err != nil {
		return planning.StoryArc{}, err
	}
	return toArc(row), nil
}

func (s *PlanningStore) NextArcVersion(ctx context.Context, arcID string) (int, error) {
	n, err := s.q.NextArcVersion(ctx, toUUID(arcID))
	return int(n), err
}

func (s *PlanningStore) CreateArcVersion(ctx context.Context, v planning.ArcVersion) (planning.ArcVersion, error) {
	content, _ := json.Marshal(v.Content)
	row, err := s.q.CreateArcVersion(ctx, db.CreateArcVersionParams{
		ID:        toUUID(v.ID),
		ArcID:     toUUID(v.ArcID),
		VersionNo: int32(v.VersionNo),
		Content:   content,
		CreatedBy: toUUID(v.CreatedBy),
	})
	if err != nil {
		return planning.ArcVersion{}, err
	}
	return toArcVersion(row), nil
}

func (s *PlanningStore) GetArc(ctx context.Context, arcID string) (planning.StoryArc, error) {
	row, err := s.q.GetArc(ctx, toUUID(arcID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return planning.StoryArc{}, planning.ErrArcNotFound
		}
		return planning.StoryArc{}, err
	}
	return toArc(row), nil
}

func (s *PlanningStore) ListArcs(ctx context.Context, storyID string) ([]planning.StoryArc, error) {
	rows, err := s.q.ListArcs(ctx, toUUID(storyID))
	if err != nil {
		return nil, err
	}
	out := make([]planning.StoryArc, 0, len(rows))
	for _, r := range rows {
		out = append(out, toArc(r))
	}
	return out, nil
}

// ============================================================
// Characters
// ============================================================

func (s *PlanningStore) CreateCharacter(ctx context.Context, c planning.Character) (planning.Character, error) {
	row, err := s.q.CreateCharacter(ctx, db.CreateCharacterParams{
		ID:            toUUID(c.ID),
		StoryID:       toUUID(c.StoryID),
		CanonicalName: c.CanonicalName,
		Importance:    c.Importance,
	})
	if err != nil {
		return planning.Character{}, err
	}
	return toCharacter(row), nil
}

func (s *PlanningStore) NextProfileVersion(ctx context.Context, characterID string) (int, error) {
	n, err := s.q.NextProfileVersion(ctx, toUUID(characterID))
	return int(n), err
}

func (s *PlanningStore) CreateProfileVersion(ctx context.Context, v planning.CharacterProfileVersion) (planning.CharacterProfileVersion, error) {
	profile, _ := json.Marshal(v.Profile)
	row, err := s.q.CreateProfileVersion(ctx, db.CreateProfileVersionParams{
		ID:          toUUID(v.ID),
		CharacterID: toUUID(v.CharacterID),
		VersionNo:   int32(v.VersionNo),
		Profile:     profile,
		CreatedBy:   toUUID(v.CreatedBy),
	})
	if err != nil {
		return planning.CharacterProfileVersion{}, err
	}
	return toProfileVersion(row), nil
}

func (s *PlanningStore) ListCharacters(ctx context.Context, storyID string) ([]planning.Character, error) {
	rows, err := s.q.ListCharacters(ctx, toUUID(storyID))
	if err != nil {
		return nil, err
	}
	out := make([]planning.Character, 0, len(rows))
	for _, r := range rows {
		out = append(out, toCharacter(r))
	}
	return out, nil
}

func (s *PlanningStore) GetCharacter(ctx context.Context, characterID string) (planning.Character, error) {
	row, err := s.q.GetCharacter(ctx, toUUID(characterID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return planning.Character{}, planning.ErrCharacterNotFound
		}
		return planning.Character{}, err
	}
	return toCharacter(row), nil
}

// ============================================================
// Converters
// ============================================================

func toBibleVersion(row db.StoryBibleVersion) planning.StoryBibleVersion {
	var content map[string]any
	_ = json.Unmarshal(row.Content, &content)
	return planning.StoryBibleVersion{
		ID:               fromUUID(row.ID),
		StoryID:          fromUUID(row.StoryID),
		VersionNo:        int(row.VersionNo),
		Content:          content,
		BasedOnVersionID: fromUUID(row.BasedOnVersionID),
		CreatedBy:        fromUUID(row.CreatedBy),
	}
}

func toEndingVersion(row db.StoryEndingPlanVersion) planning.EndingPlanVersion {
	var content map[string]any
	_ = json.Unmarshal(row.Content, &content)
	return planning.EndingPlanVersion{
		ID:               fromUUID(row.ID),
		StoryID:          fromUUID(row.StoryID),
		VersionNo:        int(row.VersionNo),
		Content:          content,
		BasedOnVersionID: fromUUID(row.BasedOnVersionID),
		CreatedBy:        fromUUID(row.CreatedBy),
	}
}

func toArc(row db.StoryArc) planning.StoryArc {
	return planning.StoryArc{
		ID:               fromUUID(row.ID),
		StoryID:          fromUUID(row.StoryID),
		Ordinal:          int(row.Ordinal),
		Status:           row.Status,
		CurrentVersionID: fromUUID(row.CurrentVersionID),
	}
}

func toArcVersion(row db.StoryArcVersion) planning.ArcVersion {
	var content map[string]any
	_ = json.Unmarshal(row.Content, &content)
	return planning.ArcVersion{
		ID:        fromUUID(row.ID),
		ArcID:     fromUUID(row.ArcID),
		VersionNo: int(row.VersionNo),
		Content:   content,
		CreatedBy: fromUUID(row.CreatedBy),
	}
}

func toCharacter(row db.Character) planning.Character {
	return planning.Character{
		ID:                     fromUUID(row.ID),
		StoryID:                fromUUID(row.StoryID),
		CanonicalName:          row.CanonicalName,
		Importance:             row.Importance,
		CurrentProfileVersionID: fromUUID(row.CurrentProfileVersionID),
	}
}

func toProfileVersion(row db.CharacterProfileVersion) planning.CharacterProfileVersion {
	var profile map[string]any
	_ = json.Unmarshal(row.Profile, &profile)
	return planning.CharacterProfileVersion{
		ID:          fromUUID(row.ID),
		CharacterID: fromUUID(row.CharacterID),
		VersionNo:   int(row.VersionNo),
		Profile:     profile,
		CreatedBy:   fromUUID(row.CreatedBy),
	}
}
