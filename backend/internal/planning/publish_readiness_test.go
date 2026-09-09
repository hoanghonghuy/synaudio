package planning

import (
	"context"
	"testing"
)

type fakePublishContentAuthority struct {
	hasApproved bool
	err         error
}

func (f *fakePublishContentAuthority) HasApprovedContent(ctx context.Context, chapterID string) (bool, error) {
	return f.hasApproved, f.err
}

type fakePublishAudioAuthority struct {
	missing []string
	err     error
}

func (f *fakePublishAudioAuthority) CheckPublishAudioReady(ctx context.Context, chapterID string) ([]string, error) {
	return f.missing, f.err
}

type fakePublishStoryAuthority struct {
	missing []string
	err     error
}

func (f *fakePublishStoryAuthority) CheckStoryPermitsChapterPublish(ctx context.Context, storyID string) ([]string, error) {
	return f.missing, f.err
}

func TestCompositePublishCheckerReportsAllMissingDependencies(t *testing.T) {
	store := newPublishFakeStore()
	ch, _ := store.CreateChapter(context.Background(), Chapter{ID: "ch-1", StoryID: "s1", Status: "READY"})

	checker := NewCompositePublishChecker(
		store,
		&fakePublishContentAuthority{hasApproved: false},
		&fakePublishAudioAuthority{missing: []string{"active_audio"}},
		&fakePublishStoryAuthority{missing: []string{"story_status"}},
	)

	missing, err := checker.CheckPublishReady(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("check publish ready: %v", err)
	}
	if len(missing) != 3 {
		t.Fatalf("expected 3 missing deps, got %v", missing)
	}
}

func TestCompositePublishCheckerReadyWhenAllDependenciesPresent(t *testing.T) {
	store := newPublishFakeStore()
	ch, _ := store.CreateChapter(context.Background(), Chapter{ID: "ch-1", StoryID: "s1", Status: "READY"})

	checker := NewCompositePublishChecker(
		store,
		&fakePublishContentAuthority{hasApproved: true},
		&fakePublishAudioAuthority{},
		&fakePublishStoryAuthority{},
	)

	missing, err := checker.CheckPublishReady(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("check publish ready: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("expected no missing deps, got %v", missing)
	}
}

func TestCheckPublishReadinessFailsClosedWithoutAuthority(t *testing.T) {
	store := newPublishFakeStore()
	svc := NewService(store)
	ch, _ := store.CreateChapter(context.Background(), Chapter{ID: "ch-1", StoryID: "s1", Status: "READY"})

	got, err := svc.CheckPublishReadiness(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("check publish readiness: %v", err)
	}
	if got.Ready {
		t.Fatal("expected not ready without publish authority")
	}
	if len(got.Missing) == 0 || got.Missing[0] != "publish_authority" {
		t.Fatalf("expected publish_authority missing, got %v", got.Missing)
	}
}

func TestCheckPublishReadinessReturnsActionableMissingDependencies(t *testing.T) {
	store := newPublishFakeStore()
	ch, _ := store.CreateChapter(context.Background(), Chapter{ID: "ch-1", StoryID: "s1", Status: "READY"})
	svc := NewService(store, WithPublishChecker(NewCompositePublishChecker(
		store,
		&fakePublishContentAuthority{hasApproved: false},
		&fakePublishAudioAuthority{missing: []string{"active_audio"}},
		&fakePublishStoryAuthority{},
	)))

	got, err := svc.CheckPublishReadiness(context.Background(), ch.ID)
	if err != nil {
		t.Fatalf("check publish readiness: %v", err)
	}
	if got.Ready {
		t.Fatal("expected blocked readiness")
	}
	if len(got.Missing) != 2 {
		t.Fatalf("expected 2 missing deps, got %v", got.Missing)
	}
}
