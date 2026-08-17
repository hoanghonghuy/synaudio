package planning

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrAttentionItemNotFound = errors.New("attention item not found")
)

// AttentionItem is an in-app attention center entry.
type AttentionItem struct {
	ID        string
	StoryID   string
	ChapterID string
	Priority  string
	Kind      string
	Title     string
	Detail    string
	Action    string
	Resolved  bool
}

// AttentionItemInput is the input for creating an attention item.
type AttentionItemInput struct {
	StoryID   string
	ChapterID string
	Priority  string
	Kind      string
	Title     string
	Detail    string
	Action    string
}

// CreateAttentionItem creates a new unresolved attention item.
func (s *Service) CreateAttentionItem(ctx context.Context, in AttentionItemInput) (AttentionItem, error) {
	if in.Title == "" {
		return AttentionItem{}, errors.New("title must not be empty")
	}
	if in.StoryID == "" {
		return AttentionItem{}, errors.New("story id must not be empty")
	}

	priority := in.Priority
	if priority == "" {
		priority = "INFORMATIONAL"
	}

	a := AttentionItem{
		ID:        uuid.NewString(),
		StoryID:   in.StoryID,
		ChapterID: in.ChapterID,
		Priority:  priority,
		Kind:      in.Kind,
		Title:     in.Title,
		Detail:    in.Detail,
		Action:    in.Action,
		Resolved:  false,
	}

	return s.store.CreateAttentionItem(ctx, a)
}

// ListAttentionItems returns all attention items for a story.
func (s *Service) ListAttentionItems(ctx context.Context, storyID string) ([]AttentionItem, error) {
	return s.store.ListAttentionItems(ctx, storyID)
}

// ResolveAttentionItem marks an attention item as resolved.
func (s *Service) ResolveAttentionItem(ctx context.Context, id string) (AttentionItem, error) {
	a, err := s.store.GetAttentionItem(ctx, id)
	if err != nil {
		return AttentionItem{}, err
	}

	a.Resolved = true

	return s.store.UpdateAttentionItem(ctx, a)
}
