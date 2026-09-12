package pgstore

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/synaudio/synaudio/backend/internal/planning"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

const (
	postponeDecisionID = "11111111-1111-1111-1111-111111111111"
	postponeStoryID    = "22222222-2222-2222-2222-222222222222"
	postponeActorID    = "33333333-3333-3333-3333-333333333333"
)

func TestUpdateCreativeDecisionPersistsPostponeContext(t *testing.T) {
	pool := newCreativeDecisionTestPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
INSERT INTO creative_decisions (
    id, story_id, origin, decision_type, severity, status, blocking_level, question
) VALUES ($1, $2, 'AI', 'PLOT', 'SIGNIFICANT', 'PROPOSED', 'NON_BLOCKING', 'Delay the reveal?')`,
		postponeDecisionID, postponeStoryID); err != nil {
		t.Fatalf("insert creative decision fixture: %v", err)
	}

	store := NewPlanningStore(db.New(pool), pool)
	updated, err := store.UpdateCreativeDecision(ctx, planning.CreativeDecision{
		ID:               postponeDecisionID,
		Status:           "POSTPONED",
		SelectedBy:       postponeActorID,
		RevisitCondition: map[string]any{"reason": "Wait until the next arc"},
	})
	if err != nil {
		t.Fatalf("update creative decision: %v", err)
	}
	if updated.Status != "POSTPONED" || updated.SelectedBy != postponeActorID {
		t.Fatalf("unexpected updated decision: %+v", updated)
	}
	if updated.RevisitCondition["reason"] != "Wait until the next arc" {
		t.Fatalf("postpone reason missing from update result: %#v", updated.RevisitCondition)
	}

	stored, err := store.GetCreativeDecision(ctx, postponeDecisionID)
	if err != nil {
		t.Fatalf("get creative decision: %v", err)
	}
	if stored.Status != "POSTPONED" || stored.SelectedBy != postponeActorID {
		t.Fatalf("unexpected persisted decision: %+v", stored)
	}
	if stored.RevisitCondition["reason"] != "Wait until the next arc" {
		t.Fatalf("postpone reason did not round-trip through PostgreSQL: %#v", stored.RevisitCondition)
	}
}

func newCreativeDecisionTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for PostgreSQL creative-decision regression tests")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	schema := fmt.Sprintf("p2_149_postpone_%d", time.Now().UnixNano())
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
CREATE TABLE creative_decisions (
    id UUID PRIMARY KEY,
    story_id UUID NOT NULL,
    chapter_id UUID,
    arc_id UUID,
    origin TEXT NOT NULL,
    decision_type TEXT NOT NULL,
    severity TEXT NOT NULL,
    status TEXT NOT NULL,
    blocking_level TEXT NOT NULL,
    question TEXT NOT NULL,
    context_summary TEXT,
    recommended_option_id UUID,
    selected_option_id UUID,
    custom_selected_text TEXT,
    rejection_scope TEXT,
    revisit_condition JSONB,
    triggered_by_run_id UUID,
    created_by UUID,
    selected_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    selected_at TIMESTAMPTZ,
    applied_at TIMESTAMPTZ
);`); err != nil {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		admin.Close(ctx)
		t.Fatalf("create creative decision table: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		admin.Close(context.Background())
	})
	return pool
}
