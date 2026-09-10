# Production observability

Synaudio exposes Prometheus-compatible metrics without replacing the existing health/readiness contract.

## Endpoints and exposure boundary

- API metrics are disabled unless `API_METRICS_ADDR` is explicitly configured. The metrics server accepts only explicit loopback/private IP binds; wildcard, public-IP and DNS-name binds are rejected.
- Worker metrics are served on `WORKER_METRICS_ADDR`, defaulting to `127.0.0.1:9091`, through the same private-bind validation.
- Worker deployment probes use `WORKER_PROBE_ADDR` (default `127.0.0.1:8081`) with `GET /health` (liveness) and `GET /ready` (database + loop heartbeat freshness). See `docs/operations/production-deployment.md`.
- `/health` and `/ready` remain on the normal product API listener and keep their existing semantics.

Metrics are diagnostic/operator surfaces, not public product endpoints. Production scrape targets should remain on the private monitoring network.

## Cardinality and privacy

Metrics use bounded labels only. Do not add user IDs, story/chapter/job IDs, raw URLs, query strings, prompts, generated content, email addresses, auth/recovery tokens, provider error messages, or action links as labels.

API request metrics use the chi route pattern after routing, not the raw request path. Unknown/invalid routes collapse to `unmatched`. Worker loop, result, queue, job-type and error-class labels are explicit allowlists and collapse unknown values to `other`/`UNKNOWN`.

## Metrics

- `synaudio_api_requests_total{method,route,status_class}`: API traffic and status-class trend.
- `synaudio_api_request_duration_seconds_sum{method,route,status_class}`: cumulative request latency. Divide by the matching request count for mean latency.
- `synaudio_auth_throttled_total{route,dimension}`: auth abuse throttles by bounded route (`POST /login`, `POST /password/forgot`, `POST /re-auth`, etc.) and dimension (`client`, `account`, `session`).
- `synaudio_worker_heartbeat_unixtime`: last observed worker loop heartbeat.
- `synaudio_worker_loop_runs_total{loop,outcome}`: success/failure of generation polling, stale reclaim, audit delivery, transactional email delivery and account-deletion reconciliation.
- `synaudio_worker_loop_items_total{loop,result}`: bounded item outcomes including reclaimed, processed, claimed, delivered, retrying, dead-letter and purged.
- `synaudio_generation_jobs_total{job_type,outcome,error_class}`: bounded generation outcome/error-class signal.
- `synaudio_generation_attempt_duration_seconds_sum{job_type,outcome,error_class}`: cumulative measured execution duration for generation attempts.
- `synaudio_provider_calls_total{provider,operation,outcome,failure_class,retry_decision}`: bounded outbound provider attempt outcomes.
- `synaudio_provider_call_duration_seconds_sum{provider,operation,outcome,failure_class,retry_decision}`: cumulative outbound provider attempt latency.
- `synaudio_backlog_depth{queue}`: current authoritative pending/retry backlog depth for `generation`, `audit_outbox`, and `email_delivery`.
- `synaudio_backlog_oldest_age_seconds{queue}`: age of the oldest current pending/retry item for each bounded queue; zero when the backlog is empty.
- `synaudio_backlog_dead_letter{queue}`: current dead-letter count where the persistence model supports dead-letter state. Generation currently reports zero because terminal generation failure is represented as `FAILED`, not a dead-letter queue.
- `synaudio_database_pool_acquired_conns{role}`: current acquired PostgreSQL pool connections (`api`, `worker`, or collapsed `other`).
- `synaudio_database_pool_idle_conns{role}`: current idle pool connections.
- `synaudio_database_pool_total_conns{role}`: current total pool connections.
- `synaudio_database_pool_max_conns{role}`: configured pool maximum for the process role.
- `synaudio_database_pool_canceled_acquires_total{role}`: cumulative acquire attempts canceled by context while waiting for a connection.
- `synaudio_database_pool_empty_acquires_total{role}`: cumulative acquires that waited because the pool was exhausted.
- `synaudio_database_pool_empty_acquire_wait_seconds_sum{role}`: cumulative wait time for exhausted-pool acquires.

Backlog gauges are sampled from durable queue tables every 15 seconds. They are current-state gauges, not values inferred from cumulative worker counters.

Database pool gauges are sampled from `pgxpool` statistics every 15 seconds on API and worker metrics listeners. See `database-pool-capacity.md` for the production connection-budget contract and saturation semantics.

## Minimum alerting/runbook

1. **Worker stale/stopped**: alert when current time minus `synaudio_worker_heartbeat_unixtime` exceeds 60 seconds for a running worker deployment. Check process/container health, DB connectivity and worker logs.
2. **Backlog age/depth**: alert when `synaudio_backlog_depth` remains non-zero while `synaudio_backlog_oldest_age_seconds` grows beyond the expected processing SLA. Split alerts by the bounded `queue` label to distinguish generation, audit and email pressure.
3. **Audit/email dead letter**: alert whenever `synaudio_backlog_dead_letter{queue="audit_outbox"}` or `{queue="email_delivery"}` is non-zero. Investigate the durable outbox before treating delivery as healthy.
4. **Retry pressure**: correlate backlog gauges with sustained growth of `audit_delivery/retrying`, stale-generation `reclaimed`, or loop `failure` counters. Inspect dependency readiness and provider logs.
5. **API 5xx trend**: compare `status_class="5xx"` request rate against total request rate per bounded route. Investigate route-specific logs and readiness dependencies.
6. **Generation/provider failures and latency**: alert on sustained increases in `synaudio_generation_jobs_total{outcome="failure"}` and on abnormal mean generation attempt duration derived from duration sum / matching outcome count. Correlate with `synaudio_provider_calls_total` using bounded `failure_class` and `retry_decision` labels. Never add raw provider error text as a metric label; use structured logs for detailed diagnosis.
7. **Database pool saturation**: alert when `synaudio_database_pool_acquired_conns` approaches `synaudio_database_pool_max_conns` for a sustained interval, or when `synaudio_database_pool_canceled_acquires_total` / `synaudio_database_pool_empty_acquires_total` grow while latency or worker loop failures increase. Correlate with the connection-budget guidance in `database-pool-capacity.md` before raising per-process `DATABASE_POOL_MAX_CONNS`.
8. **Readiness failure**: continue to use `/ready` for dependency gating. Metrics are diagnostic telemetry and do not replace readiness.

## HTTP server timeout contract (application boundary)

The API and metrics listeners set explicit `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, and `MaxHeaderBytes` values in `backend/internal/platform/httpserver` rather than relying on Go zero-value defaults or undocumented ingress behavior.

Public API defaults (as of #52):

- `ReadHeaderTimeout`: 5s
- `ReadTimeout`: 60s (request body read bound; pairs with a 2 MiB JSON body limit at the router)
- `WriteTimeout`: 11m (covers the longest synchronous TextAI admin handler budget of ~10m without leaving connections unbounded)
- `IdleTimeout`: 120s
- `MaxHeaderBytes`: 64 KiB
- `PublicMaxRequestBodyBytes`: 2 MiB

Private metrics defaults:

- `ReadHeaderTimeout`: 5s
- `ReadTimeout`: 15s
- `WriteTimeout`: 30s
- `IdleTimeout`: 60s
- `MaxHeaderBytes`: 32 KiB

Production ingress/reverse-proxy timeouts (#49) must be **at or above** these application bounds for the routes they front. If an ingress `proxy_read_timeout`/`proxy_send_timeout` is shorter than the API `WriteTimeout`, legitimate synchronous admin generation/review responses can be truncated even though the application would still be working. Metrics scrapes should use ingress/proxy timeouts no shorter than the metrics `WriteTimeout`.

## Scrape examples

Prometheus should scrape API and worker metrics targets independently over their explicitly private addresses. Do not route either metrics listener through public ingress.

When adding a new worker loop, result or queue label, update the allowlist and its tests in `backend/internal/platform/metrics` in the same PR. When adding public API routes, continue using route templates rather than raw resource paths to preserve bounded cardinality.
