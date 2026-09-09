package listener

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestSaveProgressConcurrentSameVersionOnlyOneSucceeds(t *testing.T) {
	store := newCASFakeStore()
	svc := NewService(store)

	const workers = 2
	start := make(chan struct{})
	results := make([]ListeningProgress, workers)
	errs := make([]error, workers)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			position := int64((idx + 1) * 5000)
			results[idx], errs[idx] = svc.SaveProgress(
				context.Background(),
				"u1",
				"c1",
				position,
				"asset-1",
				fmt.Sprintf("session-%d", idx),
				0,
			)
		}(i)
	}

	close(start)
	wg.Wait()

	var successes int
	var conflicts int
	for i := 0; i < workers; i++ {
		switch {
		case errs[i] == nil:
			successes++
		case errors.Is(errs[i], ErrProgressVersionConflict):
			conflicts++
		default:
			t.Fatalf("worker %d: unexpected error %v", i, errs[i])
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("expected exactly one success and one conflict, got successes=%d conflicts=%d", successes, conflicts)
	}

	current, err := svc.GetProgress(context.Background(), "u1", "c1")
	if err != nil {
		t.Fatalf("get progress: %v", err)
	}
	if current.Version != 1 {
		t.Fatalf("expected authoritative version 1, got %d", current.Version)
	}
}

func TestSaveProgressAllowsBackwardSeekWithinAuthoritativeVersion(t *testing.T) {
	store := newCASFakeStore()
	svc := NewService(store)

	first, err := svc.SaveProgress(context.Background(), "u1", "c1", 10000, "asset-1", "session-1", 0)
	if err != nil {
		t.Fatalf("first save: %v", err)
	}

	rewound, err := svc.SaveProgress(context.Background(), "u1", "c1", 2500, "asset-1", "session-1", first.Version)
	if err != nil {
		t.Fatalf("backward seek save: %v", err)
	}
	if rewound.PositionMs != 2500 {
		t.Fatalf("expected rewound position 2500, got %d", rewound.PositionMs)
	}
	if rewound.Version != first.Version+1 {
		t.Fatalf("expected version %d, got %d", first.Version+1, rewound.Version)
	}
}

func TestSaveProgressStaleWriteReturnsConflictWithCurrentState(t *testing.T) {
	store := newCASFakeStore()
	svc := NewService(store)

	first, err := svc.SaveProgress(context.Background(), "u1", "c1", 5000, "asset-1", "session-1", 0)
	if err != nil {
		t.Fatalf("seed save: %v", err)
	}
	second, err := svc.SaveProgress(context.Background(), "u1", "c1", 9000, "asset-1", "session-2", first.Version)
	if err != nil {
		t.Fatalf("authoritative save: %v", err)
	}

	_, err = svc.SaveProgress(context.Background(), "u1", "c1", 1000, "asset-1", "session-stale", first.Version)
	var conflict *ProgressVersionConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("expected ProgressVersionConflict, got %v", err)
	}
	if conflict.Current.PositionMs != second.PositionMs {
		t.Fatalf("expected conflict current position %d, got %d", second.PositionMs, conflict.Current.PositionMs)
	}
	if conflict.Current.Version != second.Version {
		t.Fatalf("expected conflict current version %d, got %d", second.Version, conflict.Current.Version)
	}
}

func TestSaveProgressDoesNotClearCompletionOnStaleWrite(t *testing.T) {
	store := newCASFakeStore()
	svc := NewService(store)

	_, err := svc.SaveProgress(context.Background(), "u1", "c1", 5000, "asset-1", "session-1", 0)
	if err != nil {
		t.Fatalf("seed save: %v", err)
	}

	completed, err := svc.MarkCompleted(context.Background(), "u1", "c1")
	if err != nil {
		t.Fatalf("mark completed: %v", err)
	}
	if completed.CompletedAt == "" {
		t.Fatal("expected completed_at to be set")
	}

	_, err = svc.SaveProgress(context.Background(), "u1", "c1", 1000, "asset-1", "session-stale", 0)
	if !errors.Is(err, ErrProgressVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}

	current, err := svc.GetProgress(context.Background(), "u1", "c1")
	if err != nil {
		t.Fatalf("get progress: %v", err)
	}
	if current.CompletedAt == "" {
		t.Fatal("stale save must not clear completion")
	}
}
