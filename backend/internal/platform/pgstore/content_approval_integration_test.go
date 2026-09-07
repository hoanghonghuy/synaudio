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

	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

const (
	approvalChapterID      = "11111111-1111-1111-1111-111111111111"
	otherApprovalChapterID = "22222222-2222-2222-2222-222222222222"
	approvalRevisionID     = "33333333-3333-3333-3333-333333333333"
	approvalActorID        = "44444444-4444-4444-4444-444444444444"
)

func TestApproveContentRevisionPersistsProjectionAndRetryIsIdempotent(t *testing.T) {
	pool := newContentApprovalTestPool(t)
	insertContentApprovalRevision(t, pool, approvalRevisionID, approvalChapterID, "CANDIDATE")
	store := NewGenerationStore(db.New(pool))
	ctx := context.Background()

	first, err := store.ApproveContentRevision(ctx, generation.ContentApproval{
		ID:                "55555555-5555-5555-5555-555555555555",
		ChapterID:         approvalChapterID,
		ContentRevisionID: approvalRevisionID,
		ApprovedBy:        approvalActorID,
	})
	if err != nil {
		t.Fatalf("first approval: %v", err)
	}
	second, err := store.ApproveContentRevision(ctx, generation.ContentApproval{
		ID:                "66666666-6666-6666-6666-666666666666",
		ChapterID:         approvalChapterID,
		ContentRevisionID: approvalRevisionID,
		ApprovedBy:        approvalActorID,
	})
	if err != nil {
		t.Fatalf("approval retry: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("retry created a second approval: first=%q second=%q", first.ID, second.ID)
	}
	assertContentApprovalState(t, pool, approvalRevisionID, "APPROVED", 1)
}

func TestApproveContentRevisionRejectsCrossChapterWithoutSideEffect(t *testing.T) {
	pool := newContentApprovalTestPool(t)
	insertContentApprovalRevision(t, pool, approvalRevisionID, approvalChapterID, "CANDIDATE")
	store := NewGenerationStore(db.New(pool))

	_, err := store.ApproveContentRevision(context.Background(), generation.ContentApproval{
		ID:                "55555555-5555-5555-5555-555555555555",
		ChapterID:         otherApprovalChapterID,
		ContentRevisionID: approvalRevisionID,
		ApprovedBy:        approvalActorID,
	})
	if !errors.Is(err, generation.ErrContentRevisionChapterMismatch) {
		t.Fatalf("expected chapter mismatch, got %v", err)
	}
	assertContentApprovalState(t, pool, approvalRevisionID, "CANDIDATE", 0)
}

func TestApproveContentRevisionRollsBackApprovalWhenProjectionFails(t *testing.T) {
	pool := newContentApprovalTestPool(t)
	insertContentApprovalRevision(t, pool, approvalRevisionID, approvalChapterID, "CANDIDATE")
	if _, err := pool.Exec(context.Background(), `
ALTER TABLE chapter_content_revisions
ADD CONSTRAINT reject_approved_projection CHECK (status <> 'APPROVED')`); err != nil {
		t.Fatalf("install projection failure constraint: %v", err)
	}
	store := NewGenerationStore(db.New(pool))

	_, err := store.ApproveContentRevision(context.Background(), generation.ContentApproval{
		ID:                "55555555-5555-5555-5555-555555555555",
		ChapterID:         approvalChapterID,
		ContentRevisionID: approvalRevisionID,
		ApprovedBy:        approvalActorID,
	})
	if err == nil {
		t.Fatal("expected projection failure")
	}
	assertContentApprovalState(t, pool, approvalRevisionID, "CANDIDATE", 0)
}

func TestApproveContentRevisionConcurrentRetriesCreateOneApproval(t *testing.T) {
	pool := newContentApprovalTestPool(t)
	insertContentApprovalRevision(t, pool, approvalRevisionID, approvalChapterID, "CANDIDATE")
	store := NewGenerationStore(db.New(pool))
	ctx := context.Background()

	start := make(chan struct{})
	type result struct {
		approval generation.ContentApproval
		err      error
	}
	results := make(chan result, 2)
	ids := []string{
		"55555555-5555-5555-5555-555555555555",
		"66666666-6666-6666-6666-666666666666",
	}
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			<-start
			approval, err := store.ApproveContentRevision(ctx, generation.ContentApproval{
				ID:                id,
				ChapterID:         approvalChapterID,
				ContentRevisionID: approvalRevisionID,
				ApprovedBy:        approvalActorID,
			})
			results <- result{approval: approval, err: err}
		}(id)
	}
	close(start)
	wg.Wait()
	close(results)

	returnedID := ""
	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent approval: %v", result.err)
		}
		if returnedID == "" {
			returnedID = result.approval.ID
		} else if result.approval.ID != returnedID {
			t.Fatalf("concurrent retries returned different approvals: %q vs %q", returnedID, result.approval.ID)
		}
	}
	assertContentApprovalState(t, pool, approvalRevisionID, "APPROVED", 1)
}

func newContentApprovalTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for PostgreSQL content-approval regression tests")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	schema := fmt.Sprintf("p0_88_%d", time.Now().UnixNano())
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
CREATE TABLE chapter_content_revisions (
    id UUID PRIMARY KEY,
    chapter_id UUID NOT NULL,
    revision_no INTEGER NOT NULL,
    content_text TEXT NOT NULL,
    source_type TEXT NOT NULL DEFAULT 'AI_GENERATED',
    based_on_revision_id UUID,
    plan_revision_id UUID,
    base_canon_version_id UUID,
    generation_run_id UUID,
    retcon_request_id UUID,
    status TEXT NOT NULL DEFAULT 'CANDIDATE',
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE content_approvals (
    id UUID PRIMARY KEY,
    chapter_id UUID NOT NULL,
    content_revision_id UUID NOT NULL,
    approved_by UUID,
    approved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    warnings_snapshot JSONB,
    override_snapshot JSONB
);`); err != nil {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		admin.Close(ctx)
		t.Fatalf("create approval tables: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		admin.Close(context.Background())
	})
	return pool
}

func insertContentApprovalRevision(t *testing.T, pool *pgxpool.Pool, revisionID, chapterID, status string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
INSERT INTO chapter_content_revisions (
    id, chapter_id, revision_no, content_text, source_type, status
) VALUES ($1, $2, 1, 'content', 'AI_GENERATED', $3)`, revisionID, chapterID, status); err != nil {
		t.Fatalf("insert content revision: %v", err)
	}
}

func assertContentApprovalState(t *testing.T, pool *pgxpool.Pool, revisionID, wantStatus string, wantApprovals int) {
	t.Helper()
	var status string
	if err := pool.QueryRow(context.Background(), `SELECT status FROM chapter_content_revisions WHERE id = $1`, revisionID).Scan(&status); err != nil {
		t.Fatalf("read revision status: %v", err)
	}
	if status != wantStatus {
		t.Fatalf("revision status=%q want %q", status, wantStatus)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM content_approvals WHERE content_revision_id = $1`, revisionID).Scan(&count); err != nil {
		t.Fatalf("count approvals: %v", err)
	}
	if count != wantApprovals {
		t.Fatalf("approval count=%d want %d", count, wantApprovals)
	}
}
