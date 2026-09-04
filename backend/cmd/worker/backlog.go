package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	platformmetrics "github.com/synaudio/synaudio/backend/internal/platform/metrics"
)

type backlogSnapshot struct {
	depth      int64
	oldest     pgtype.Timestamptz
	deadLetter int64
}

func refreshBacklogMetrics(ctx context.Context, pool *pgxpool.Pool, registry *platformmetrics.Registry, now time.Time) error {
	samples := []struct {
		queue string
		query string
	}{
		{
			queue: "generation",
			query: `SELECT COUNT(*) FILTER (WHERE status = 'PENDING'),
			               MIN(created_at) FILTER (WHERE status = 'PENDING'),
			               0::bigint
			        FROM generation_jobs`,
		},
		{
			queue: "audit_outbox",
			query: `SELECT COUNT(*) FILTER (WHERE status IN ('PENDING', 'DELIVERING')),
			               MIN(created_at) FILTER (WHERE status IN ('PENDING', 'DELIVERING')),
			               COUNT(*) FILTER (WHERE status = 'DEAD_LETTER')
			        FROM audit_delivery_outbox`,
		},
		{
			queue: "email_delivery",
			query: `SELECT COUNT(*) FILTER (WHERE status IN ('PENDING', 'DELIVERING')),
			               MIN(created_at) FILTER (WHERE status IN ('PENDING', 'DELIVERING')),
			               COUNT(*) FILTER (WHERE status = 'DEAD_LETTER')
			        FROM email_delivery_outbox`,
		},
	}

	for _, sample := range samples {
		var snapshot backlogSnapshot
		if err := pool.QueryRow(ctx, sample.query).Scan(&snapshot.depth, &snapshot.oldest, &snapshot.deadLetter); err != nil {
			return err
		}
		registry.SetBacklog(sample.queue, snapshot.depth, oldestAge(now, snapshot.oldest), snapshot.deadLetter)
	}
	return nil
}

func oldestAge(now time.Time, oldest pgtype.Timestamptz) time.Duration {
	if !oldest.Valid || oldest.Time.IsZero() || !now.After(oldest.Time) {
		return 0
	}
	return now.Sub(oldest.Time)
}
