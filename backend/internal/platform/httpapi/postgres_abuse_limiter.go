package httpapi

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

// PostgresAbuseLimiter stores counters in PostgreSQL so multi-instance API
// replicas share one authoritative limit budget.
type PostgresAbuseLimiter struct {
	queries *db.Queries
}

func NewPostgresAbuseLimiter(queries *db.Queries) *PostgresAbuseLimiter {
	return &PostgresAbuseLimiter{queries: queries}
}

func (p *PostgresAbuseLimiter) Allow(ctx context.Context, scope, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	if limit <= 0 || window <= 0 {
		return true, 0, nil
	}
	now := time.Now()
	windowStart := now.Truncate(window)
	count, err := p.queries.IncrementAuthAbuseCounter(ctx, db.IncrementAuthAbuseCounterParams{
		Scope:   scope,
		KeyHash: key,
		WindowStart: pgtype.Timestamptz{
			Time:  windowStart,
			Valid: true,
		},
	})
	if err != nil {
		return false, 0, err
	}
	if int(count) > limit {
		retryAfter := windowStart.Add(window).Sub(now)
		if retryAfter < time.Second {
			retryAfter = time.Second
		}
		return false, retryAfter, nil
	}
	return true, 0, nil
}
