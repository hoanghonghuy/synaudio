package httpapi

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter is a simple in-memory per-client rate limiter using a fixed
// window. It is intended for coarse protection of public endpoints; a
// distributed limiter (e.g. Redis) should replace it in multi-instance
// deployments.
type RateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	requests map[string][]time.Time
}

// NewRateLimiter returns a middleware that allows at most `limit` requests per
// `window` per client IP.
func NewRateLimiter(limit int, windowSeconds int) func(http.Handler) http.Handler {
	rl := &RateLimiter{
		limit:    limit,
		window:   time.Duration(windowSeconds) * time.Second,
		requests: map[string][]time.Time{},
	}
	return rl.middleware
}

func (rl *RateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.allow(ip) {
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"error": map[string]string{
					"code":    "RATE_LIMITED",
					"message": "too many requests",
				},
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allow(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	times := rl.requests[ip]
	kept := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	rl.requests[ip] = kept

	if len(kept) >= rl.limit {
		return false
	}

	rl.requests[ip] = append(kept, now)
	return true
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
