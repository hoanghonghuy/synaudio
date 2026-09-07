package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReclaimStaleJobsReconcilesInterruptedAttemptBelowBudget(t *testing.T) {
	pool := newAttemptBudgetTestPool(t)
	createAttemptReclaimTable(t, pool)
	ctx := context.Background()
	jobID := "66666666-6666-6666-6666-666666666666"
	insertAttemptBudgetJob(t, pool, jobID, "RUNNING", 2, 3, true)
	insertReclaimAttempt(t, pool, jobID, 2, "RUNNING")

	jobs, err := New(pool).ReclaimStaleJobs(ctx)
	if err != nil {
		t.Fatalf("reclaim stale job below budget: %v", err)
	}
	if len(jobs) != 1 || jobs[0].Status != "PENDING" || jobs[0].AttemptCount != 2 {
		t.Fatalf("unexpected reclaim result: %+v", jobs)
	}
	assertReclaimAttempt(t, pool, jobID, 2, "FAILED", "INTERRUPTED", "LEASE_EXPIRED", true)
}

func TestReclaimStaleJobsReconcilesInterruptedAttemptAtBudget(t *testing.T) {
	pool := newAttemptBudgetTestPool(t)
	createAttemptReclaimTable(t, pool)
	ctx := context.Background()
	jobID := "77777777-7777-7777-7777-777777777777"
	insertAttemptBudgetJob(t, pool, jobID, "RUNNING", 3, 3, true)
	insertReclaimAttempt(t, pool, jobID, 3, "RUNNING")

	jobs, err := New(pool).ReclaimStaleJobs(ctx)
	if err != nil {
		t.Fatalf("reclaim stale job at budget: %v", err)
	}
	if len(jobs) != 1 || jobs[0].Status != "FAILED" {
		t.Fatalf("unexpected reclaim result: %+v", jobs)
	}
	if !jobs[0].LastErrorClass.Valid || jobs[0].LastErrorClass.String != "RETRY_EXHAUSTED" ||
		!jobs[0].LastErrorCode.Valid || jobs[0].LastErrorCode.String != "MAX_ATTEMPTS_EXHAUSTED" {
		t.Fatalf("unexpected exhausted job authority: %+v", jobs[0])
	}
	assertReclaimAttempt(t, pool, jobID, 3, "FAILED", "INTERRUPTED", "LEASE_EXPIRED", true)
}

func TestReclaimStaleJobsDoesNotMutateLiveLeaseOrTerminalAttempt(t *testing.T) {
	pool := newAttemptBudgetTestPool(t)
	createAttemptReclaimTable(t, pool)
	ctx := context.Background()

	liveJobID := "88888888-8888-8888-8888-888888888888"
	insertAttemptBudgetJob(t, pool, liveJobID, "RUNNING", 1, 3, false)
	if _, err := pool.Exec(ctx, `UPDATE generation_jobs SET locked_by = 'live-worker', lock_expires_at = NOW() + INTERVAL '5 minutes' WHERE id = $1::uuid`, liveJobID); err != nil {
		t.Fatalf("set live lease: %v", err)
	}
	insertReclaimAttempt(t, pool, liveJobID, 1, "RUNNING")

	terminalJobID := "99999999-9999-9999-9999-999999999999"
	insertAttemptBudgetJob(t, pool, terminalJobID, "RUNNING", 1, 3, true)
	insertReclaimAttempt(t, pool, terminalJobID, 1, "SUCCEEDED")

	jobs, err := New(pool).ReclaimStaleJobs(ctx)
	if err != nil {
		t.Fatalf("reclaim stale jobs: %v", err)
	}
	if len(jobs) != 1 || jobs[0].Status != "PENDING" {
		t.Fatalf("expected only expired job to be reclaimed, got %+v", jobs)
	}
	liveStatus, liveAttempts := readAttemptBudgetJob(t, pool, liveJobID)
	if liveStatus != "RUNNING" || liveAttempts != 1 {
		t.Fatalf("live job was mutated by reclaim: status=%s attempts=%d", liveStatus, liveAttempts)
	}

	assertReclaimAttempt(t, pool, liveJobID, 1, "RUNNING", "", "", false)
	assertReclaimAttempt(t, pool, terminalJobID, 1, "SUCCEEDED", "", "", false)
}

func createAttemptReclaimTable(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
CREATE TABLE generation_job_attempts (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES generation_jobs(id) ON DELETE CASCADE,
    attempt_no INTEGER NOT NULL,
    provider TEXT,
    model TEXT,
    status TEXT NOT NULL,
    error_class TEXT,
    error_code TEXT,
    safe_error_detail TEXT,
    usage JSONB,
    latency_ms INTEGER,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    UNIQUE(job_id, attempt_no)
)`)
	if err != nil {
		t.Fatalf("create generation_job_attempts test table: %v", err)
	}
}

func insertReclaimAttempt(t *testing.T, pool *pgxpool.Pool, jobID string, attemptNo int, status string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
INSERT INTO generation_job_attempts (id, job_id, attempt_no, provider, model, status, started_at)
VALUES (gen_random_uuid(), $1::uuid, $2, 'test-provider', 'test-model', $3, NOW() - INTERVAL '2 minutes')`,
		jobID, attemptNo, status)
	if err != nil {
		t.Fatalf("insert generation attempt: %v", err)
	}
}

func assertReclaimAttempt(t *testing.T, pool *pgxpool.Pool, jobID string, attemptNo int, wantStatus, wantClass, wantCode string, wantCompleted bool) {
	t.Helper()
	var status string
	var errorClass, errorCode *string
	var completedAt *time.Time
	if err := pool.QueryRow(context.Background(), `
SELECT status, error_class, error_code, completed_at
FROM generation_job_attempts
WHERE job_id = $1::uuid AND attempt_no = $2`, jobID, attemptNo).Scan(&status, &errorClass, &errorCode, &completedAt); err != nil {
		t.Fatalf("read generation attempt: %v", err)
	}
	if status != wantStatus {
		t.Fatalf("unexpected attempt status: got %s want %s", status, wantStatus)
	}
	if wantClass == "" {
		if errorClass != nil {
			t.Fatalf("unexpected attempt error_class: %v", *errorClass)
		}
	} else if errorClass == nil || *errorClass != wantClass {
		t.Fatalf("unexpected attempt error_class: got %v want %s", errorClass, wantClass)
	}
	if wantCode == "" {
		if errorCode != nil {
			t.Fatalf("unexpected attempt error_code: %v", *errorCode)
		}
	} else if errorCode == nil || *errorCode != wantCode {
		t.Fatalf("unexpected attempt error_code: got %v want %s", errorCode, wantCode)
	}
	if wantCompleted != (completedAt != nil) {
		t.Fatalf("unexpected completed_at presence: got %v wantCompleted=%v", completedAt, wantCompleted)
	}
}
