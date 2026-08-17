package planning

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrBibleNotFound = errors.New("story bible not found")
)

// StoryBibleVersion is a versioned, relatively-stable Story Bible.
type StoryBibleVersion struct {
	ID              string
	StoryID         string
	VersionNo       int
	Content         map[string]any
	BasedOnVersionID string
	CreatedBy       string
}

// Store is the persistence boundary for the planning service.
type Store interface {
	NextBibleVersion(ctx context.Context, storyID string) (int, error)
	CreateBibleVersion(ctx context.Context, v StoryBibleVersion) (StoryBibleVersion, error)
	GetCurrentBible(ctx context.Context, storyID string) (StoryBibleVersion, error)

	NextEndingVersion(ctx context.Context, storyID string) (int, error)
	CreateEndingVersion(ctx context.Context, v EndingPlanVersion) (EndingPlanVersion, error)
	GetCurrentEnding(ctx context.Context, storyID string) (EndingPlanVersion, error)

	NextArcOrdinal(ctx context.Context, storyID string) (int, error)
	CreateArc(ctx context.Context, a StoryArc) (StoryArc, error)
	NextArcVersion(ctx context.Context, arcID string) (int, error)
	CreateArcVersion(ctx context.Context, v ArcVersion) (ArcVersion, error)
	GetArc(ctx context.Context, arcID string) (StoryArc, error)
	ListArcs(ctx context.Context, storyID string) ([]StoryArc, error)

	CreateCharacter(ctx context.Context, c Character) (Character, error)
	NextProfileVersion(ctx context.Context, characterID string) (int, error)
	CreateProfileVersion(ctx context.Context, v CharacterProfileVersion) (CharacterProfileVersion, error)
	ListCharacters(ctx context.Context, storyID string) ([]Character, error)
	GetCharacter(ctx context.Context, characterID string) (Character, error)

	NextChapterNumber(ctx context.Context, storyID string) (int, error)
	CreateChapter(ctx context.Context, c Chapter) (Chapter, error)
	NextPlanRevision(ctx context.Context, chapterID string) (int, error)
	CreatePlanRevision(ctx context.Context, p ChapterPlanRevision) (ChapterPlanRevision, error)
	GetChapter(ctx context.Context, chapterID string) (Chapter, error)
	ListChapters(ctx context.Context, storyID string) ([]Chapter, error)

	CreateFact(ctx context.Context, f StoryFact) (StoryFact, error)
	ListFacts(ctx context.Context, storyID string) ([]StoryFact, error)

	CreatePlotThread(ctx context.Context, t PlotThread) (PlotThread, error)
	ListPlotThreads(ctx context.Context, storyID string) ([]PlotThread, error)
	CreatePlotThreadEvent(ctx context.Context, e PlotThreadEvent) (PlotThreadEvent, error)
	ListPlotThreadEvents(ctx context.Context, threadID string) ([]PlotThreadEvent, error)

	CreateCanonBranch(ctx context.Context, b CanonBranch) (CanonBranch, error)
	NextCanonSequence(ctx context.Context, branchID string) (int, error)
	CreateCanonVersion(ctx context.Context, v CanonVersion) (CanonVersion, error)
	ListCanonVersions(ctx context.Context, branchID string) ([]CanonVersion, error)
	GetCanonVersion(ctx context.Context, id string) (CanonVersion, error)
	UpdateCanonVersion(ctx context.Context, v CanonVersion) (CanonVersion, error)
	CreateCanonChangeItem(ctx context.Context, c CanonChangeItem) (CanonChangeItem, error)
	ListCanonChangeItems(ctx context.Context, canonVersionID string) ([]CanonChangeItem, error)

	CreateContextSnapshot(ctx context.Context, sn ContextSnapshot) (ContextSnapshot, error)
	ListContextSnapshots(ctx context.Context, storyID string) ([]ContextSnapshot, error)
	GetContextSnapshot(ctx context.Context, id string) (ContextSnapshot, error)

	UpdateChapterStatus(ctx context.Context, chapterID, status string) (Chapter, error)

	CreateCreativeDecision(ctx context.Context, d CreativeDecision) (CreativeDecision, error)
	GetCreativeDecision(ctx context.Context, id string) (CreativeDecision, error)
	ListCreativeDecisions(ctx context.Context, storyID string) ([]CreativeDecision, error)
	UpdateCreativeDecision(ctx context.Context, d CreativeDecision) (CreativeDecision, error)

	CreateAttentionItem(ctx context.Context, a AttentionItem) (AttentionItem, error)
	ListAttentionItems(ctx context.Context, storyID string) ([]AttentionItem, error)
	GetAttentionItem(ctx context.Context, id string) (AttentionItem, error)
	UpdateAttentionItem(ctx context.Context, a AttentionItem) (AttentionItem, error)
}

type Service struct {
	store           Store
	architect       Architect
	memoryExtractor MemoryExtractor
	publishChecker  PublishChecker
}

type Option func(*Service)

func WithArchitect(a Architect) Option {
	return func(svc *Service) {
		svc.architect = a
	}
}

func WithMemoryExtractor(m MemoryExtractor) Option {
	return func(svc *Service) {
		svc.memoryExtractor = m
	}
}

func WithPublishChecker(c PublishChecker) Option {
	return func(svc *Service) {
		svc.publishChecker = c
	}
}

func NewService(store Store, opts ...Option) *Service {
	svc := &Service{store: store}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// CreateBibleVersion creates a new versioned Story Bible.
func (s *Service) CreateBibleVersion(ctx context.Context, storyID string, content map[string]any, createdBy string) (StoryBibleVersion, error) {
	if len(content) == 0 {
		return StoryBibleVersion{}, errors.New("bible content must not be empty")
	}

	versionNo, err := s.store.NextBibleVersion(ctx, storyID)
	if err != nil {
		return StoryBibleVersion{}, err
	}

	v := StoryBibleVersion{
		ID:        uuid.NewString(),
		StoryID:   storyID,
		VersionNo: versionNo,
		Content:   content,
		CreatedBy: createdBy,
	}

	return s.store.CreateBibleVersion(ctx, v)
}

// GetCurrentBible returns the latest Story Bible version.
func (s *Service) GetCurrentBible(ctx context.Context, storyID string) (StoryBibleVersion, error) {
	return s.store.GetCurrentBible(ctx, storyID)
}
