package httpapi

import (
	"context"
	"sync"
	"time"
)

// AbuseLimiter enforces fixed-window counters for auth abuse scopes.
type AbuseLimiter interface {
	Allow(ctx context.Context, scope string, key string, limit int, window time.Duration) (allowed bool, retryAfter time.Duration, err error)
}

type memoryCounter struct {
	windowStart time.Time
	count       int
}

type MemoryAbuseLimiter struct {
	mu       sync.Mutex
	counters map[string]memoryCounter
}

func NewMemoryAbuseLimiter() *MemoryAbuseLimiter {
	return &MemoryAbuseLimiter{counters: map[string]memoryCounter{}}
}

func (m *MemoryAbuseLimiter) Allow(_ context.Context, scope, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	if limit <= 0 || window <= 0 {
		return true, 0, nil
	}
	now := time.Now()
	windowStart := now.Truncate(window)
	counterKey := scope + "\x00" + key + "\x00" + windowStart.UTC().Format(time.RFC3339Nano)

	m.mu.Lock()
	defer m.mu.Unlock()

	counter := m.counters[counterKey]
	if counter.windowStart != windowStart {
		counter = memoryCounter{windowStart: windowStart, count: 0}
	}
	counter.count++
	m.counters[counterKey] = counter

	if counter.count > limit {
		retryAfter := windowStart.Add(window).Sub(now)
		if retryAfter < time.Second {
			retryAfter = time.Second
		}
		return false, retryAfter, nil
	}
	return true, 0, nil
}
