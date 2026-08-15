package listener

import (
	"context"
	"testing"
)

func TestAddFavoriteIsIdempotent(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if err := svc.AddFavorite(context.Background(), "u1", "s1"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := svc.AddFavorite(context.Background(), "u1", "s1"); err != nil {
		t.Fatalf("add again: %v", err)
	}

	favs, err := svc.ListFavorites(context.Background(), "u1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(favs) != 1 {
		t.Fatalf("expected 1 favorite, got %d", len(favs))
	}
}

func TestRemoveFavorite(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	_ = svc.AddFavorite(context.Background(), "u1", "s1")
	if err := svc.RemoveFavorite(context.Background(), "u1", "s1"); err != nil {
		t.Fatalf("remove: %v", err)
	}

	favs, _ := svc.ListFavorites(context.Background(), "u1")
	if len(favs) != 0 {
		t.Fatalf("expected 0 favorites, got %d", len(favs))
	}
}

func TestIsFavorite(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	_ = svc.AddFavorite(context.Background(), "u1", "s1")

	isFav, err := svc.IsFavorite(context.Background(), "u1", "s1")
	if err != nil {
		t.Fatalf("is favorite: %v", err)
	}
	if !isFav {
		t.Fatal("expected favorite")
	}

	isFav2, _ := svc.IsFavorite(context.Background(), "u1", "s2")
	if isFav2 {
		t.Fatal("expected not favorite")
	}
}

func TestSaveProgressCreatesAndUpdates(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	p1, err := svc.SaveProgress(context.Background(), "u1", "c1", 5000, "asset-1", "session-1")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if p1.PositionMs != 5000 {
		t.Fatalf("expected 5000, got %d", p1.PositionMs)
	}

	p2, err := svc.SaveProgress(context.Background(), "u1", "c1", 10000, "asset-1", "session-1")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if p2.PositionMs != 10000 {
		t.Fatalf("expected 10000, got %d", p2.PositionMs)
	}
	if p2.Version <= p1.Version {
		t.Fatalf("expected version increment, got %d -> %d", p1.Version, p2.Version)
	}
}

func TestGetProgressReturnsNotFound(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.GetProgress(context.Background(), "u1", "c1"); err == nil {
		t.Fatal("expected error for missing progress")
	}
}

func TestMarkCompleted(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	_, _ = svc.SaveProgress(context.Background(), "u1", "c1", 5000, "asset-1", "session-1")

	completed, err := svc.MarkCompleted(context.Background(), "u1", "c1")
	if err != nil {
		t.Fatalf("mark completed: %v", err)
	}
	if completed.CompletedAt == "" {
		t.Fatal("expected completed_at to be set")
	}
}
