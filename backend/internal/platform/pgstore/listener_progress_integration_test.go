package pgstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/synaudio/synaudio/backend/internal/listener"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

const (
	progressUserID    = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	progressChapterID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

func TestSaveProgressConcurrentSameVersionPostgreSQL(t *testing.T) {
	pool := newListenerProgressTestPool(t)
	ctx := context.Background()
	store := NewListenerStore(db.New(pool))

	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make([]listener.ListeningProgress, 2)
	errs := make([]error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			position := int64((idx + 1) * 7000)
			results[idx], errs[idx] = store.SaveProgress(ctx, listener.ListeningProgress{
				UserID:                progressUserID,
				ChapterID:             progressChapterID,
				PositionMs:            position,
				LastAudioAssetID:      "cccccccc-cccc-cccc-cccc-cccccccccccc",
				LastPlaybackSessionID: fmt.Sprintf("dddddddd-dddd-dddd-dddd-dddddddddd%02d", idx),
			}, 0)
		}(i)
	}

	close(start)
	wg.Wait()

	var successes int
	var conflicts int
	for i := 0; i < 2; i++ {
		switch {
		case errs[i] == nil:
			successes++
		case errors.As(errs[i], new(*listener.ProgressVersionConflict)):
			conflicts++
		default:
			t.Fatalf("worker %d: unexpected error %v", i, errs[i])
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("expected one success and one conflict, got successes=%d conflicts=%d", successes, conflicts)
	}

	current, err := store.GetProgress(ctx, progressUserID, progressChapterID)
	if err != nil {
		t.Fatalf("get progress: %v", err)
	}
	if current.Version != 1 {
		t.Fatalf("expected authoritative version 1, got %d", current.Version)
	}
}

func TestSaveProgressPreservesCompletionOnStaleWritePostgreSQL(t *testing.T) {
	pool := newListenerProgressTestPool(t)
	ctx := context.Background()
	store := NewListenerStore(db.New(pool))

	_, err := store.SaveProgress(ctx, listener.ListeningProgress{
		UserID:                progressUserID,
		ChapterID:             progressChapterID,
		PositionMs:            5000,
		LastAudioAssetID:      "cccccccc-cccc-cccc-cccc-cccccccccccc",
		LastPlaybackSessionID: "dddddddd-dddd-dddd-dddd-dddddddddd01",
	}, 0)
	if err != nil {
		t.Fatalf("seed save: %v", err)
	}

	completed, err := store.MarkCompleted(ctx, progressUserID, progressChapterID)
	if err != nil {
		t.Fatalf("mark completed: %v", err)
	}
	if completed.CompletedAt == "" {
		t.Fatal("expected completed_at")
	}

	_, err = store.SaveProgress(ctx, listener.ListeningProgress{
		UserID:                progressUserID,
		ChapterID:             progressChapterID,
		PositionMs:            1000,
		LastAudioAssetID:      "cccccccc-cccc-cccc-cccc-cccccccccccc",
		LastPlaybackSessionID: "dddddddd-dddd-dddd-dddd-dddddddddd99",
	}, 0)
	if !errors.As(err, new(*listener.ProgressVersionConflict)) {
		t.Fatalf("expected version conflict, got %v", err)
	}

	current, err := store.GetProgress(ctx, progressUserID, progressChapterID)
	if err != nil {
		t.Fatalf("get progress: %v", err)
	}
	if current.CompletedAt == "" {
		t.Fatal("stale save must not clear completion")
	}
}

func newListenerProgressTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for PostgreSQL listener-progress regression tests")
	}

	ctx := context.Background()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	schema := fmt.Sprintf("fix_12_progress_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close(ctx)
		t.Fatalf("create test schema: %v", err)
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		admin.Close(ctx)
		t.Fatalf("parse postgres config: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		admin.Close(ctx)
		t.Fatalf("create test pool: %v", err)
	}

	if _, err := pool.Exec(ctx, `
CREATE TABLE listening_progress (
    user_id UUID NOT NULL,
    chapter_id UUID NOT NULL,
    position_ms BIGINT NOT NULL DEFAULT 0,
    completed_at TIMESTAMPTZ,
    last_audio_asset_id UUID,
    last_playback_session_id UUID,
    version BIGINT NOT NULL DEFAULT 0,
    relisten_status TEXT NOT NULL DEFAULT 'NO_RELISTEN_NEEDED',
    last_listened_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, chapter_id)
);`); err != nil {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		admin.Close(ctx)
		t.Fatalf("create listening_progress table: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		admin.Close(context.Background())
	})
	return pool
}
