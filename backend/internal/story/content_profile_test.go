package story_test

import (
	"context"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/story"
)

func TestCreateContentProfileVersionStartsAtOne(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A"}

	in := story.ContentProfileInput{
		MaturityTarget: "TEEN",
		AllowedThemes:  []string{"adventure", "friendship"},
	}

	cp, err := svc.CreateContentProfileVersion(context.Background(), "s1", in)
	if err != nil {
		t.Fatalf("create content profile: %v", err)
	}
	if cp.VersionNo != 1 {
		t.Fatalf("expected version 1, got %d", cp.VersionNo)
	}
	if cp.Profile["maturity_target"] != "TEEN" {
		t.Fatalf("expected maturity_target TEEN, got %v", cp.Profile["maturity_target"])
	}
}

func TestCreateContentProfileVersionIncrements(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A"}

	in := story.ContentProfileInput{MaturityTarget: "TEEN"}
	if _, err := svc.CreateContentProfileVersion(context.Background(), "s1", in); err != nil {
		t.Fatalf("first create: %v", err)
	}

	cp, err := svc.CreateContentProfileVersion(context.Background(), "s1", in)
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if cp.VersionNo != 2 {
		t.Fatalf("expected version 2, got %d", cp.VersionNo)
	}
}

func TestGetCurrentContentProfileReturnsLatest(t *testing.T) {
	store := newFakeStore()
	svc := story.NewService(store)

	store.stories["s1"] = story.Story{ID: "s1", Slug: "a", Title: "A"}

	in := story.ContentProfileInput{MaturityTarget: "TEEN"}
	if _, err := svc.CreateContentProfileVersion(context.Background(), "s1", in); err != nil {
		t.Fatalf("first create: %v", err)
	}
	in2 := story.ContentProfileInput{MaturityTarget: "MATURE"}
	if _, err := svc.CreateContentProfileVersion(context.Background(), "s1", in2); err != nil {
		t.Fatalf("second create: %v", err)
	}

	cp, err := svc.GetCurrentContentProfile(context.Background(), "s1")
	if err != nil {
		t.Fatalf("get current: %v", err)
	}
	if cp.Profile["maturity_target"] != "MATURE" {
		t.Fatalf("expected MATURE, got %v", cp.Profile["maturity_target"])
	}
}
