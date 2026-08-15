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
// Chapters
// ============================================================

func (s *PlanningStore) NextChapterNumber(ctx context.Context, storyID string) (int, error) {
	n, err := s.q.NextChapterNumber(ctx, toUUID(storyID))
	return int(n), err
}

func (s *PlanningStore) CreateChapter(ctx context.Context, c planning.Chapter) (planning.Chapter, error) {
	row, err := s.q.CreateChapter(ctx, db.CreateChapterParams{
		ID:            toUUID(c.ID),
		StoryID:       toUUID(c.StoryID),
		ChapterNumber: int32(c.ChapterNumber),
		Title:         toText(c.Title),
		Status:        c.Status,
		ArcID:         toUUID(c.ArcID),
	})
	if err != nil {
		return planning.Chapter{}, err
	}
	return toChapter(row), nil
}

func (s *PlanningStore) NextPlanRevision(ctx context.Context, chapterID string) (int, error) {
	n, err := s.q.NextPlanRevision(ctx, toUUID(chapterID))
	return int(n), err
}

func (s *PlanningStore) CreatePlanRevision(ctx context.Context, p planning.ChapterPlanRevision) (planning.ChapterPlanRevision, error) {
	plan, _ := json.Marshal(p.Plan)
	row, err := s.q.CreatePlanRevision(ctx, db.CreatePlanRevisionParams{
		ID:         toUUID(p.ID),
		ChapterID:  toUUID(p.ChapterID),
		RevisionNo: int32(p.RevisionNo),
		Plan:       plan,
		SourceType: p.SourceType,
		CreatedBy:  toUUID(p.CreatedBy),
	})
	if err != nil {
		return planning.ChapterPlanRevision{}, err
	}
	return toPlanRevision(row), nil
}

func (s *PlanningStore) GetChapter(ctx context.Context, chapterID string) (planning.Chapter, error) {
	row, err := s.q.GetChapter(ctx, toUUID(chapterID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return planning.Chapter{}, planning.ErrChapterNotFound
		}
		return planning.Chapter{}, err
	}
	return toChapter(row), nil
}

func (s *PlanningStore) ListChapters(ctx context.Context, storyID string) ([]planning.Chapter, error) {
	rows, err := s.q.ListChapters(ctx, toUUID(storyID))
	if err != nil {
		return nil, err
	}
	out := make([]planning.Chapter, 0, len(rows))
	for _, r := range rows {
		out = append(out, toChapter(r))
	}
	return out, nil
}

// ============================================================
// StoryFacts
// ============================================================

func (s *PlanningStore) CreateFact(ctx context.Context, f planning.StoryFact) (planning.StoryFact, error) {
	value, _ := json.Marshal(f.Value)
	row, err := s.q.CreateFact(ctx, db.CreateFactParams{
		ID:          toUUID(f.ID),
		StoryID:     toUUID(f.StoryID),
		SubjectType: toText(f.SubjectType),
		SubjectID:   toUUID(f.SubjectID),
		FactType:    f.FactType,
		Value:       value,
		Importance:  f.Importance,
		Status:      f.Status,
	})
	if err != nil {
		return planning.StoryFact{}, err
	}
	return toFact(row), nil
}

func (s *PlanningStore) ListFacts(ctx context.Context, storyID string) ([]planning.StoryFact, error) {
	rows, err := s.q.ListFacts(ctx, toUUID(storyID))
	if err != nil {
		return nil, err
	}
	out := make([]planning.StoryFact, 0, len(rows))
	for _, r := range rows {
		out = append(out, toFact(r))
	}
	return out, nil
}

// ============================================================
// PlotThreads
// ============================================================

func (s *PlanningStore) CreatePlotThread(ctx context.Context, t planning.PlotThread) (planning.PlotThread, error) {
	row, err := s.q.CreatePlotThread(ctx, db.CreatePlotThreadParams{
		ID:         toUUID(t.ID),
		StoryID:    toUUID(t.StoryID),
		Title:      t.Title,
		Summary:    toText(t.Summary),
		Importance: t.Importance,
		Status:     t.Status,
	})
	if err != nil {
		return planning.PlotThread{}, err
	}
	return toPlotThread(row), nil
}

func (s *PlanningStore) ListPlotThreads(ctx context.Context, storyID string) ([]planning.PlotThread, error) {
	rows, err := s.q.ListPlotThreads(ctx, toUUID(storyID))
	if err != nil {
		return nil, err
	}
	out := make([]planning.PlotThread, 0, len(rows))
	for _, r := range rows {
		out = append(out, toPlotThread(r))
	}
	return out, nil
}

func (s *PlanningStore) CreatePlotThreadEvent(ctx context.Context, e planning.PlotThreadEvent) (planning.PlotThreadEvent, error) {
	detail, _ := json.Marshal(e.Detail)
	row, err := s.q.CreatePlotThreadEvent(ctx, db.CreatePlotThreadEventParams{
		ID:           toUUID(e.ID),
		PlotThreadID: toUUID(e.PlotThreadID),
		ChapterID:    toUUID(e.ChapterID),
		EventType:    e.EventType,
		Detail:       detail,
	})
	if err != nil {
		return planning.PlotThreadEvent{}, err
	}
	return toPlotThreadEvent(row), nil
}

// ============================================================
// Canon branches + versions
// ============================================================

func (s *PlanningStore) CreateCanonBranch(ctx context.Context, b planning.CanonBranch) (planning.CanonBranch, error) {
	row, err := s.q.CreateCanonBranch(ctx, db.CreateCanonBranchParams{
		ID:      toUUID(b.ID),
		StoryID: toUUID(b.StoryID),
		Type:    b.Type,
		Status:  b.Status,
	})
	if err != nil {
		return planning.CanonBranch{}, err
	}
	return toCanonBranch(row), nil
}

func (s *PlanningStore) NextCanonSequence(ctx context.Context, branchID string) (int, error) {
	n, err := s.q.NextCanonSequence(ctx, toUUID(branchID))
	return int(n), err
}

func (s *PlanningStore) CreateCanonVersion(ctx context.Context, v planning.CanonVersion) (planning.CanonVersion, error) {
	row, err := s.q.CreateCanonVersion(ctx, db.CreateCanonVersionParams{
		ID:              toUUID(v.ID),
		StoryID:         toUUID(v.StoryID),
		BranchID:        toUUID(v.BranchID),
		SequenceNo:      int32(v.SequenceNo),
		SourceChapterID: toUUID(v.SourceChapterID),
		Status:          v.Status,
		CommittedBy:     toUUID(v.CommittedBy),
	})
	if err != nil {
		return planning.CanonVersion{}, err
	}
	return toCanonVersion(row), nil
}

func (s *PlanningStore) ListCanonVersions(ctx context.Context, branchID string) ([]planning.CanonVersion, error) {
	rows, err := s.q.ListCanonVersions(ctx, toUUID(branchID))
	if err != nil {
		return nil, err
	}
	out := make([]planning.CanonVersion, 0, len(rows))
	for _, r := range rows {
		out = append(out, toCanonVersion(r))
	}
	return out, nil
}

// ============================================================
// ContextSnapshots
// ============================================================

func (s *PlanningStore) CreateContextSnapshot(ctx context.Context, sn planning.ContextSnapshot) (planning.ContextSnapshot, error) {
	row, err := s.q.CreateContextSnapshot(ctx, db.CreateContextSnapshotParams{
		ID:                      toUUID(sn.ID),
		StoryID:                 toUUID(sn.StoryID),
		ChapterID:               toUUID(sn.ChapterID),
		BibleVersionID:          toUUID(sn.BibleVersionID),
		EndingPlanVersionID:     toUUID(sn.EndingPlanVersionID),
		ArcVersionID:            toUUID(sn.ArcVersionID),
		ContentProfileVersionID: toUUID(sn.ContentProfileVersionID),
		PromptVersion:           toText(sn.PromptVersion),
		WorkflowVersion:         toText(sn.WorkflowVersion),
		Provider:                toText(sn.Provider),
		Model:                   toText(sn.Model),
	})
	if err != nil {
		return planning.ContextSnapshot{}, err
	}
	return toContextSnapshot(row), nil
}

func (s *PlanningStore) ListContextSnapshots(ctx context.Context, storyID string) ([]planning.ContextSnapshot, error) {
	rows, err := s.q.ListContextSnapshots(ctx, toUUID(storyID))
	if err != nil {
		return nil, err
	}
	out := make([]planning.ContextSnapshot, 0, len(rows))
	for _, r := range rows {
		out = append(out, toContextSnapshot(r))
	}
	return out, nil
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

func toChapter(row db.Chapter) planning.Chapter {
	return planning.Chapter{
		ID:            fromUUID(row.ID),
		StoryID:       fromUUID(row.StoryID),
		ChapterNumber: int(row.ChapterNumber),
		Title:         fromText(row.Title),
		Status:        row.Status,
		ArcID:         fromUUID(row.ArcID),
	}
}

func toPlanRevision(row db.ChapterPlanRevision) planning.ChapterPlanRevision {
	var plan map[string]any
	_ = json.Unmarshal(row.Plan, &plan)
	return planning.ChapterPlanRevision{
		ID:         fromUUID(row.ID),
		ChapterID:  fromUUID(row.ChapterID),
		RevisionNo: int(row.RevisionNo),
		Plan:       plan,
		SourceType: row.SourceType,
		CreatedBy:  fromUUID(row.CreatedBy),
	}
}

func toFact(row db.StoryFact) planning.StoryFact {
	var value map[string]any
	_ = json.Unmarshal(row.Value, &value)
	return planning.StoryFact{
		ID:          fromUUID(row.ID),
		StoryID:     fromUUID(row.StoryID),
		SubjectType: fromText(row.SubjectType),
		SubjectID:   fromUUID(row.SubjectID),
		FactType:    row.FactType,
		Value:       value,
		Importance:  row.Importance,
		Status:      row.Status,
	}
}

func toPlotThread(row db.PlotThread) planning.PlotThread {
	return planning.PlotThread{
		ID:         fromUUID(row.ID),
		StoryID:    fromUUID(row.StoryID),
		Title:      row.Title,
		Summary:    fromText(row.Summary),
		Importance: row.Importance,
		Status:     row.Status,
	}
}

func toPlotThreadEvent(row db.PlotThreadEvent) planning.PlotThreadEvent {
	var detail map[string]any
	_ = json.Unmarshal(row.Detail, &detail)
	return planning.PlotThreadEvent{
		ID:           fromUUID(row.ID),
		PlotThreadID: fromUUID(row.PlotThreadID),
		ChapterID:    fromUUID(row.ChapterID),
		EventType:    row.EventType,
		Detail:       detail,
	}
}

func toCanonBranch(row db.CanonBranch) planning.CanonBranch {
	return planning.CanonBranch{
		ID:      fromUUID(row.ID),
		StoryID: fromUUID(row.StoryID),
		Type:    row.Type,
		Status:  row.Status,
	}
}

func toCanonVersion(row db.CanonVersion) planning.CanonVersion {
	return planning.CanonVersion{
		ID:              fromUUID(row.ID),
		StoryID:         fromUUID(row.StoryID),
		BranchID:        fromUUID(row.BranchID),
		SequenceNo:      int(row.SequenceNo),
		ParentVersionID: fromUUID(row.ParentVersionID),
		SourceChapterID: fromUUID(row.SourceChapterID),
		Status:          row.Status,
		CommittedBy:     fromUUID(row.CommittedBy),
	}
}

func toContextSnapshot(row db.ContextSnapshot) planning.ContextSnapshot {
	return planning.ContextSnapshot{
		ID:                      fromUUID(row.ID),
		StoryID:                 fromUUID(row.StoryID),
		ChapterID:               fromUUID(row.ChapterID),
		BibleVersionID:          fromUUID(row.BibleVersionID),
		EndingPlanVersionID:     fromUUID(row.EndingPlanVersionID),
		ArcVersionID:            fromUUID(row.ArcVersionID),
		ContentProfileVersionID: fromUUID(row.ContentProfileVersionID),
		PromptVersion:           fromText(row.PromptVersion),
		WorkflowVersion:         fromText(row.WorkflowVersion),
		Provider:                fromText(row.Provider),
		Model:                   fromText(row.Model),
	}
}
