package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestClaimNextJobFinalAllowedAttempt(t *testing.T) {
	pool := newAttemptBudgetTestPool(t)
	ctx := context.Background()
	insertAttemptBudgetJob(t, pool, "11111111-1111-1111-1111-111111111111", "PENDING", 2, 3, false)

	job, err := New(pool).ClaimNextJob(ctx, pgtype.Text{String: "worker-a", Valid: true})
	if err != nil {
		t.Fatalf("claim final allowed attempt: %v", err)
	}
	if job.Status != "RUNNING" || job.AttemptCount != 3 || job.MaxAttempts != 3 {
		t.Fatalf("unexpected claimed job: status=%s attempts=%d/%d", job.Status, job.AttemptCount, job.MaxAttempts)
	}
}

func TestClaimNextJobRejectsExhaustedPendingJob(t *testing.T) {
	pool := newAttemptBudgetTestPool(t)
	ctx := context.Background()
	jobID := "22222222-2222-2222-2222-222222222222"
	insertAttemptBudgetJob(t, pool, jobID, "PENDING", 3, 3, false)

	_, err := New(pool).ClaimNextJob(ctx, pgtype.Text{String: "worker-a", Valid: true})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected pgx.ErrNoRows for exhausted pending job, got %v", err)
	}

	status, attempts := readAttemptBudgetJob(t, pool, jobID)
	if status != "PENDING" || attempts != 3 {
		t.Fatalf("exhausted pending job mutated: status=%s attempts=%d", status, attempts)
	}
}

func TestReclaimStaleJobsBelowBudgetReturnsPending(t *testing.T) {
	pool := newAttemptBudgetTestPool(t)
	ctx := context.Background()
	jobID := "33333333-3333-3333-3333-333333333333"
	insertAttemptBudgetJob(t, pool, jobID, "RUNNING", 2, 3, true)

	jobs, err := New(pool).ReclaimStaleJobs(ctx)
	if err != nil {
		t.Fatalf("reclaim stale job below budget: %v", err)
	}
	if len(jobs) != 1 || jobs[0].Status != "PENDING" || jobs[0].AttemptCount != 2 {
		t.Fatalf("unexpected reclaim result: %+v", jobs)
	}

	status, attempts := readAttemptBudgetJob(t, pool, jobID)
	if status != "PENDING" || attempts != 2 {
		t.Fatalf("below-budget stale job not preserved: status=%s attempts=%d", status, attempts)
	}
}

func TestReclaimStaleJobsAtBudgetFailsTerminally(t *testing.T) {
	pool := newAttemptBudgetTestPool(t)
	ctx := context.Background()
	jobID := "44444444-4444-4444-4444-444444444444"
	insertAttemptBudgetJob(t, pool, jobID, "RUNNING", 3, 3, true)

	jobs, err := New(pool).ReclaimStaleJobs(ctx)
	if err != nil {
		t.Fatalf("reclaim exhausted stale job: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected one reclaimed job, got %d", len(jobs))
	}
	job := jobs[0]
	if job.Status != "FAILED" || job.AttemptCount != 3 {
		t.Fatalf("unexpected exhausted reclaim: status=%s attempts=%d", job.Status, job.AttemptCount)
	}
	if !job.LastErrorClass.Valid || job.LastErrorClass.String != "RETRY_EXHAUSTED" {
		t.Fatalf("unexpected error class: %+v", job.LastErrorClass)
	}
	if !job.LastErrorCode.Valid || job.LastErrorCode.String != "MAX_ATTEMPTS_EXHAUSTED" {
		t.Fatalf("unexpected error code: %+v", job.LastErrorCode)
	}
	if !job.CompletedAt.Valid || job.LockedBy.Valid || job.LockExpiresAt.Valid {
		t.Fatalf("terminal exhaustion did not clear lease/stamp completion: %+v", job)
	}
}

func TestClaimNextJobConcurrentNearAttemptLimit(t *testing.T) {
	pool := newAttemptBudgetTestPool(t)
	ctx := context.Background()
	jobID := "55555555-5555-5555-5555-555555555555"
	insertAttemptBudgetJob(t, pool, jobID, "PENDING", 2, 3, false)

	queries := New(pool)
	start := make(chan struct{})
	type result struct {
		job GenerationJob
		err error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for _, worker := range []string{"worker-a", "worker-b"} {
		wg.Add(1)
		go func(worker string) {
			defer wg.Done()
			<-start
			job, err := queries.ClaimNextJob(ctx, pgtype.Text{String: worker, Valid: true})
			results <- result{job: job, err: err}
		}(worker)
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	noRows := 0
	for result := range results {
		switch {
		case result.err == nil:
			successes++
			if result.job.AttemptCount != 3 || result.job.Status != "RUNNING" {
				t.Fatalf("winning claim exceeded/failed final attempt invariant: %+v", result.job)
			}
		case errors.Is(result.err, pgx.ErrNoRows):
			noRows++
		default:
			t.Fatalf("unexpected concurrent claim error: %v", result.err)
		}
	}
	if successes != 1 || noRows != 1 {
		t.Fatalf("expected exactly one admitted claim and one rejection, successes=%d noRows=%d", successes, noRows)
	}

	status, attempts := readAttemptBudgetJob(t, pool, jobID)
	if status != "RUNNING" || attempts != 3 {
		t.Fatalf("concurrent claims exceeded budget: status=%s attempts=%d", status, attempts)
	}
}

func newAttemptBudgetTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for PostgreSQL attempt-budget regression tests")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	schema := fmt.Sprintf("task44_%d", time.Now().UnixNano())
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
CREATE TABLE generation_jobs (
    id UUID PRIMARY KEY,
    run_id UUID NOT NULL,
    job_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    priority INTEGER NOT NULL DEFAULT 0,
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    input_fingerprint TEXT,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    locked_by TEXT,
    lock_expires_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    last_error_class TEXT,
    last_error_code TEXT,
    output_ref JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`); err != nil {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		admin.Close(ctx)
		t.Fatalf("create generation_jobs test table: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		admin.Close(context.Background())
	})
	return pool
}

func insertAttemptBudgetJob(t *testing.T, pool *pgxpool.Pool, id, status string, attemptCount, maxAttempts int, staleLease bool) {
	t.Helper()
	var lockedBy any
	var lockExpiresAt any
	if staleLease {
		lockedBy = "stale-worker"
		lockExpiresAt = time.Now().Add(-time.Minute)
	}
	_, err := pool.Exec(context.Background(), `
INSERT INTO generation_jobs (
    id, run_id, job_type, status, priority, available_at, attempt_count, max_attempts,
    locked_by, lock_expires_at, created_at
) VALUES ($1::uuid, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'::uuid, 'WRITER', $2, 0, NOW() - INTERVAL '1 minute', $3, $4, $5, $6, NOW())`,
		id, status, attemptCount, maxAttempts, lockedBy, lockExpiresAt)
	if err != nil {
		t.Fatalf("insert generation job: %v", err)
	}
}

func readAttemptBudgetJob(t *testing.T, pool *pgxpool.Pool, id string) (string, int32) {
	t.Helper()
	var status string
	var attempts int32
	if err := pool.QueryRow(context.Background(),
		"SELECT status, attempt_count FROM generation_jobs WHERE id = $1::uuid", id,
	).Scan(&status, &attempts); err != nil {
		t.Fatalf("read generation job: %v", err)
	}
	return status, attempts
}
