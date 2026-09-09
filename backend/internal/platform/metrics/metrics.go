package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

type backlogGauge struct {
	depth            int64
	oldestAgeSeconds float64
	deadLetter       int64
}

type Registry struct {
	mu sync.RWMutex

	apiRequests map[string]uint64
	apiDuration map[string]float64

	workerHeartbeat    int64
	workerLoopRuns     map[string]uint64
	workerLoopItems    map[string]uint64
	generationJobs     map[string]uint64
	generationDuration map[string]float64
	backlogs           map[string]backlogGauge
	databasePools      map[string]databasePoolSnapshot
}

func NewRegistry() *Registry {
	return &Registry{
		apiRequests:        make(map[string]uint64),
		apiDuration:        make(map[string]float64),
		workerLoopRuns:     make(map[string]uint64),
		workerLoopItems:    make(map[string]uint64),
		generationJobs:     make(map[string]uint64),
		generationDuration: make(map[string]float64),
		backlogs:      make(map[string]backlogGauge),
		databasePools: make(map[string]databasePoolSnapshot),
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(p)
}

func (r *Registry) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, req)
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		route := chi.RouteContext(req.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		r.ObserveAPI(req.Method, route, status, time.Since(started))
	})
}

func (r *Registry) ObserveAPI(method, route string, status int, duration time.Duration) {
	method = boundedMethod(method)
	route = boundedRoute(route)
	statusClass := strconv.Itoa(status/100) + "xx"
	key := strings.Join([]string{method, route, statusClass}, "\x00")

	r.mu.Lock()
	r.apiRequests[key]++
	r.apiDuration[key] += duration.Seconds()
	r.mu.Unlock()
}

func (r *Registry) WorkerHeartbeat(at time.Time) {
	r.mu.Lock()
	r.workerHeartbeat = at.Unix()
	r.mu.Unlock()
}

// HeartbeatAge returns how long ago the worker loop last refreshed its heartbeat.
// When no heartbeat has been recorded yet, it returns a duration larger than any
// production readiness threshold so startup probes fail closed until the loop runs.
func (r *Registry) HeartbeatAge(now time.Time) time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.workerHeartbeat == 0 {
		return 24 * time.Hour
	}
	return now.Sub(time.Unix(r.workerHeartbeat, 0))
}

func (r *Registry) ObserveWorkerLoop(loop string, err error) {
	outcome := "success"
	if err != nil {
		outcome = "failure"
	}
	key := boundedLoop(loop) + "\x00" + outcome
	r.mu.Lock()
	r.workerLoopRuns[key]++
	r.mu.Unlock()
}

func (r *Registry) AddWorkerItems(loop, result string, n int) {
	if n <= 0 {
		return
	}
	key := boundedLoop(loop) + "\x00" + boundedResult(result)
	r.mu.Lock()
	r.workerLoopItems[key] += uint64(n)
	r.mu.Unlock()
}

func (r *Registry) ObserveGenerationJob(jobType, outcome, errorClass string) {
	key := generationKey(jobType, outcome, errorClass)
	r.mu.Lock()
	r.generationJobs[key]++
	r.mu.Unlock()
}

func (r *Registry) ObserveGenerationDuration(jobType, outcome, errorClass string, duration time.Duration) {
	key := generationKey(jobType, outcome, errorClass)
	r.mu.Lock()
	r.generationDuration[key] += duration.Seconds()
	r.mu.Unlock()
}

func (r *Registry) SetBacklog(queue string, depth int64, oldestAge time.Duration, deadLetter int64) {
	queue = boundedQueue(queue)
	if depth < 0 {
		depth = 0
	}
	if oldestAge < 0 {
		oldestAge = 0
	}
	if deadLetter < 0 {
		deadLetter = 0
	}
	r.mu.Lock()
	r.backlogs[queue] = backlogGauge{depth: depth, oldestAgeSeconds: oldestAge.Seconds(), deadLetter: deadLetter}
	r.mu.Unlock()
}

func generationKey(jobType, outcome, errorClass string) string {
	return strings.Join([]string{boundedJobType(jobType), boundedOutcome(outcome), boundedErrorClass(errorClass)}, "\x00")
}

func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		r.writePrometheus(w)
	})
}

func (r *Registry) writePrometheus(w http.ResponseWriter) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, _ = fmt.Fprintln(w, "# HELP synaudio_api_requests_total API requests by bounded route, method, and status class.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_api_requests_total counter")
	for _, key := range sortedKeys(r.apiRequests) {
		parts := strings.Split(key, "\x00")
		_, _ = fmt.Fprintf(w, "synaudio_api_requests_total{method=%q,route=%q,status_class=%q} %d\n", parts[0], parts[1], parts[2], r.apiRequests[key])
	}

	_, _ = fmt.Fprintln(w, "# HELP synaudio_api_request_duration_seconds_sum Cumulative API request duration by bounded route, method, and status class.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_api_request_duration_seconds_sum counter")
	for _, key := range sortedKeys(r.apiDuration) {
		parts := strings.Split(key, "\x00")
		_, _ = fmt.Fprintf(w, "synaudio_api_request_duration_seconds_sum{method=%q,route=%q,status_class=%q} %g\n", parts[0], parts[1], parts[2], r.apiDuration[key])
	}

	_, _ = fmt.Fprintln(w, "# HELP synaudio_worker_heartbeat_unixtime Last worker heartbeat Unix timestamp.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_worker_heartbeat_unixtime gauge")
	_, _ = fmt.Fprintf(w, "synaudio_worker_heartbeat_unixtime %d\n", r.workerHeartbeat)

	_, _ = fmt.Fprintln(w, "# HELP synaudio_worker_loop_runs_total Worker loop executions by bounded loop and outcome.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_worker_loop_runs_total counter")
	for _, key := range sortedKeys(r.workerLoopRuns) {
		parts := strings.Split(key, "\x00")
		_, _ = fmt.Fprintf(w, "synaudio_worker_loop_runs_total{loop=%q,outcome=%q} %d\n", parts[0], parts[1], r.workerLoopRuns[key])
	}

	_, _ = fmt.Fprintln(w, "# HELP synaudio_worker_loop_items_total Worker loop item outcomes such as reclaimed, delivered, retrying, or dead_letter.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_worker_loop_items_total counter")
	for _, key := range sortedKeys(r.workerLoopItems) {
		parts := strings.Split(key, "\x00")
		_, _ = fmt.Fprintf(w, "synaudio_worker_loop_items_total{loop=%q,result=%q} %d\n", parts[0], parts[1], r.workerLoopItems[key])
	}

	_, _ = fmt.Fprintln(w, "# HELP synaudio_generation_jobs_total Generation job processing outcomes using bounded job/error labels.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_generation_jobs_total counter")
	for _, key := range sortedKeys(r.generationJobs) {
		parts := strings.Split(key, "\x00")
		_, _ = fmt.Fprintf(w, "synaudio_generation_jobs_total{job_type=%q,outcome=%q,error_class=%q} %d\n", parts[0], parts[1], parts[2], r.generationJobs[key])
	}

	_, _ = fmt.Fprintln(w, "# HELP synaudio_generation_attempt_duration_seconds_sum Cumulative generation attempt execution duration using bounded job/outcome/error labels.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_generation_attempt_duration_seconds_sum counter")
	for _, key := range sortedKeys(r.generationDuration) {
		parts := strings.Split(key, "\x00")
		_, _ = fmt.Fprintf(w, "synaudio_generation_attempt_duration_seconds_sum{job_type=%q,outcome=%q,error_class=%q} %g\n", parts[0], parts[1], parts[2], r.generationDuration[key])
	}

	_, _ = fmt.Fprintln(w, "# HELP synaudio_backlog_depth Current authoritative pending/retry backlog depth by bounded queue.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_backlog_depth gauge")
	_, _ = fmt.Fprintln(w, "# HELP synaudio_backlog_oldest_age_seconds Age in seconds of the oldest pending/retry item by bounded queue.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_backlog_oldest_age_seconds gauge")
	_, _ = fmt.Fprintln(w, "# HELP synaudio_backlog_dead_letter Current dead-letter item count by bounded queue.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_backlog_dead_letter gauge")
	for _, queue := range sortedBacklogKeys(r.backlogs) {
		gauge := r.backlogs[queue]
		_, _ = fmt.Fprintf(w, "synaudio_backlog_depth{queue=%q} %d\n", queue, gauge.depth)
		_, _ = fmt.Fprintf(w, "synaudio_backlog_oldest_age_seconds{queue=%q} %g\n", queue, gauge.oldestAgeSeconds)
		_, _ = fmt.Fprintf(w, "synaudio_backlog_dead_letter{queue=%q} %d\n", queue, gauge.deadLetter)
	}

	_, _ = fmt.Fprintln(w, "# HELP synaudio_database_pool_acquired_conns Current acquired PostgreSQL pool connections by bounded process role.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_database_pool_acquired_conns gauge")
	_, _ = fmt.Fprintln(w, "# HELP synaudio_database_pool_idle_conns Current idle PostgreSQL pool connections by bounded process role.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_database_pool_idle_conns gauge")
	_, _ = fmt.Fprintln(w, "# HELP synaudio_database_pool_total_conns Current total PostgreSQL pool connections by bounded process role.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_database_pool_total_conns gauge")
	_, _ = fmt.Fprintln(w, "# HELP synaudio_database_pool_max_conns Configured PostgreSQL pool maximum by bounded process role.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_database_pool_max_conns gauge")
	_, _ = fmt.Fprintln(w, "# HELP synaudio_database_pool_canceled_acquires_total Cumulative canceled pool acquire attempts by bounded process role.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_database_pool_canceled_acquires_total counter")
	_, _ = fmt.Fprintln(w, "# HELP synaudio_database_pool_empty_acquires_total Cumulative pool acquires that waited for a connection by bounded process role.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_database_pool_empty_acquires_total counter")
	_, _ = fmt.Fprintln(w, "# HELP synaudio_database_pool_empty_acquire_wait_seconds_sum Cumulative wait time for empty pool acquires by bounded process role.")
	_, _ = fmt.Fprintln(w, "# TYPE synaudio_database_pool_empty_acquire_wait_seconds_sum counter")
	for _, role := range sortedKeysString(r.databasePools) {
		snapshot := r.databasePools[role]
		_, _ = fmt.Fprintf(w, "synaudio_database_pool_acquired_conns{role=%q} %d\n", role, snapshot.acquiredConns)
		_, _ = fmt.Fprintf(w, "synaudio_database_pool_idle_conns{role=%q} %d\n", role, snapshot.idleConns)
		_, _ = fmt.Fprintf(w, "synaudio_database_pool_total_conns{role=%q} %d\n", role, snapshot.totalConns)
		_, _ = fmt.Fprintf(w, "synaudio_database_pool_max_conns{role=%q} %d\n", role, snapshot.maxConns)
		_, _ = fmt.Fprintf(w, "synaudio_database_pool_canceled_acquires_total{role=%q} %d\n", role, snapshot.canceledAcquires)
		_, _ = fmt.Fprintf(w, "synaudio_database_pool_empty_acquires_total{role=%q} %d\n", role, snapshot.emptyAcquires)
		_, _ = fmt.Fprintf(w, "synaudio_database_pool_empty_acquire_wait_seconds_sum{role=%q} %g\n", role, snapshot.emptyAcquireWaitSec)
	}
}

func sortedKeysString[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func boundedMethod(v string) string {
	switch v {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions:
		return v
	default:
		return "OTHER"
	}
}

func boundedRoute(v string) string {
	if v == "" || len(v) > 160 || strings.ContainsAny(v, "?\n\r") {
		return "unmatched"
	}
	return v
}

func boundedLoop(v string) string {
	switch v {
	case "generation_poll", "stale_reclaim", "audit_delivery", "email_delivery", "account_deletion":
		return v
	default:
		return "other"
	}
}

func boundedResult(v string) string {
	switch v {
	case "claimed", "delivered", "retrying", "dead_letter", "reclaimed", "processed", "purged":
		return v
	default:
		return "other"
	}
}

func boundedQueue(v string) string {
	switch v {
	case "generation", "audit_outbox", "email_delivery":
		return v
	default:
		return "other"
	}
}

func boundedJobType(v string) string {
	switch v {
	case "WRITER":
		return v
	default:
		return "UNKNOWN"
	}
}

func boundedOutcome(v string) string {
	switch v {
	case "success", "failure", "no_work":
		return v
	default:
		return "other"
	}
}

func boundedErrorClass(v string) string {
	switch v {
	case "", "TRANSIENT", "PERMANENT", "UNKNOWN_JOB_TYPE":
		if v == "" {
			return "none"
		}
		return v
	default:
		return "other"
	}
}

func sortedKeys[V ~uint64 | ~float64](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedBacklogKeys(m map[string]backlogGauge) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
