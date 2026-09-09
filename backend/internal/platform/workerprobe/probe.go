package workerprobe

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// DefaultMaxHeartbeatAge matches the worker stale alert threshold documented in
// docs/operations/observability.md and the production deployment runbook.
const DefaultMaxHeartbeatAge = 60 * time.Second

// Dependencies configures the worker-specific liveness/readiness contract.
// Readiness proves the worker process can reach PostgreSQL and its main loop is
// emitting fresh heartbeats. External AI/TTS providers are intentionally excluded
// so transient provider outages remain observable through job metrics without
// failing worker readiness or API liveness.
type Dependencies struct {
	PingDatabase    func(context.Context) error
	HeartbeatAge    func() time.Duration
	AcceptingWork   func() bool
	MaxHeartbeatAge time.Duration
}

func maxHeartbeatAge(deps Dependencies) time.Duration {
	if deps.MaxHeartbeatAge > 0 {
		return deps.MaxHeartbeatAge
	}
	return DefaultMaxHeartbeatAge
}

// NewHandler exposes GET /health and GET /ready for orchestrator probes.
func NewHandler(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		status := http.StatusOK
		body := map[string]any{"status": "ready"}
		depStatus := map[string]string{}

		if deps.AcceptingWork != nil && !deps.AcceptingWork() {
			status = http.StatusServiceUnavailable
			body["status"] = "draining"
			body["error"] = "worker_draining"
		}

		if deps.PingDatabase != nil {
			pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := deps.PingDatabase(pingCtx); err != nil {
				depStatus["database"] = "unavailable"
				status = http.StatusServiceUnavailable
				body["status"] = "degraded"
				body["error"] = "dependency_unavailable"
			} else {
				depStatus["database"] = "ok"
			}
		}

		if deps.HeartbeatAge != nil {
			age := deps.HeartbeatAge()
			maxAge := maxHeartbeatAge(deps)
			if age > maxAge {
				depStatus["worker_loop"] = "stale"
				status = http.StatusServiceUnavailable
				if body["status"] == "ready" {
					body["status"] = "degraded"
				}
				body["error"] = "worker_loop_stale"
			} else {
				depStatus["worker_loop"] = "ok"
			}
		}

		if len(depStatus) > 0 {
			body["dependencies"] = depStatus
		}

		writeJSON(w, status, body)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
