package metrics

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type databasePoolSnapshot struct {
	acquiredConns       int32
	idleConns           int32
	totalConns          int32
	maxConns            int32
	canceledAcquires    int64
	emptyAcquires       int64
	emptyAcquireWaitSec float64
}

type databasePoolStatsSource interface {
	Stat() *pgxpool.Stat
}

// SetDatabasePool records a bounded snapshot of pgxpool saturation signals.
func (r *Registry) SetDatabasePool(role string, stat *pgxpool.Stat) {
	if stat == nil {
		return
	}
	role = boundedDatabasePoolRole(role)
	r.mu.Lock()
	if r.databasePools == nil {
		r.databasePools = make(map[string]databasePoolSnapshot)
	}
	r.databasePools[role] = databasePoolSnapshot{
		acquiredConns:       stat.AcquiredConns(),
		idleConns:           stat.IdleConns(),
		totalConns:          stat.TotalConns(),
		maxConns:            stat.MaxConns(),
		canceledAcquires:    stat.CanceledAcquireCount(),
		emptyAcquires:       stat.EmptyAcquireCount(),
		emptyAcquireWaitSec: stat.EmptyAcquireWaitTime().Seconds(),
	}
	r.mu.Unlock()
}

// StartDatabasePoolSampler publishes pool saturation gauges on a fixed interval.
func StartDatabasePoolSampler(ctx context.Context, pool databasePoolStatsSource, registry *Registry, role string, interval time.Duration) {
	if pool == nil || registry == nil {
		return
	}
	if interval <= 0 {
		interval = 15 * time.Second
	}

	sample := func() {
		registry.SetDatabasePool(role, pool.Stat())
	}
	sample()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sample()
			}
		}
	}()
}

func boundedDatabasePoolRole(v string) string {
	switch v {
	case "api", "worker":
		return v
	default:
		return "other"
	}
}
