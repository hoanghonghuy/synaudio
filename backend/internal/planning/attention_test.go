package planning

import (
	"context"
	"testing"
)

func TestCreateAttentionItem(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	item, err := svc.CreateAttentionItem(context.Background(), AttentionItemInput{
		StoryID:  "s1",
		Priority: "BLOCKING",
		Kind:     "CREATIVE_DECISION",
		Title:    "Decision required",
		Detail:   "A major decision needs admin input",
		Action:   "open-decision",
	})
	if err != nil {
		t.Fatalf("create attention item: %v", err)
	}
	if item.Priority != "BLOCKING" {
		t.Fatalf("expected BLOCKING, got %q", item.Priority)
	}
	if item.Resolved {
		t.Fatal("expected unresolved item")
	}
}

func TestCreateAttentionItemRejectsEmptyTitle(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.CreateAttentionItem(context.Background(), AttentionItemInput{
		StoryID: "s1",
	}); err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestListAttentionItemsUnresolved(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	_, _ = svc.CreateAttentionItem(context.Background(), AttentionItemInput{
		StoryID: "s1", Priority: "WARNING", Kind: "GEN", Title: "A", Action: "x",
	})
	_, _ = svc.CreateAttentionItem(context.Background(), AttentionItemInput{
		StoryID: "s1", Priority: "BLOCKING", Kind: "GEN", Title: "B", Action: "x",
	})

	items, err := svc.ListAttentionItems(context.Background(), "s1")
	if err != nil {
		t.Fatalf("list attention items: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestResolveAttentionItem(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	item, _ := svc.CreateAttentionItem(context.Background(), AttentionItemInput{
		StoryID: "s1", Priority: "WARNING", Kind: "GEN", Title: "A", Action: "x",
	})

	resolved, err := svc.ResolveAttentionItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("resolve attention item: %v", err)
	}
	if !resolved.Resolved {
		t.Fatal("expected resolved item")
	}
}
