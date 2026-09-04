# Production observability

Synaudio exposes Prometheus-compatible metrics without replacing the existing health/readiness contract.

## Endpoints and exposure boundary

- API metrics are disabled unless `API_METRICS_ADDR` is explicitly configured. The metrics server accepts only explicit loopback/private IP binds; wildcard, public-IP and DNS-name binds are rejected.
- Worker metrics are served on `WORKER_METRICS_ADDR`, defaulting to `127.0.0.1:9091`, through the same private-bind validation.
- `/health` and `/ready` remain on the normal product API listener and keep their existing semantics.

Metrics are diagnostic/operator surfaces, not public product endpoints. Production scrape targets should remain on the private monitoring network.

## Cardinality and privacy

Metrics use bounded labels only. Do not add user IDs, story/chapter/job IDs, raw URLs, query strings, prompts, generated content, email addresses, auth/recovery tokens, provider error messages, or action links as labels.

API request metrics use the chi route pattern after routing, not the raw request path. Unknown/invalid routes collapse to `unmatched`. Worker loop, result, queue, job-type and error-class labels are explicit allowlists and collapse unknown values to `other`/`UNKNOWN`.

## Metrics

- `synaudio_api_requests_total{method,route,status_class}`: API traffic and status-class trend.
- `synaudio_api_request_duration_seconds_sum{method,route,status_class}`: cumulative request latency. Divide by the matching request count for mean latency.
- `synaudio_worker_heartbeat_unixtime`: last observed worker loop heartbeat.
- `synaudio_worker_loop_runs_total{loop,outcome}`: success/failure of generation polling, stale reclaim, audit delivery, transactional email delivery and account-deletion reconciliation.
- `synaudio_worker_loop_items_total{loop,result}`: bounded item outcomes including reclaimed, processed, claimed, delivered, retrying, dead-letter and purged.
- `synaudio_generation_jobs_total{job_type,outcome,error_class}`: bounded generation outcome/error-class signal.
- `synaudio_generation_attempt_duration_seconds_sum{job_type,outcome,error_class}`: cumulative measured execution duration for generation attempts.
- `synaudio_backlog_depth{queue}`: current authoritative pending/retry backlog depth for `generation`, `audit_outbox`, and `email_delivery`.
- `synaudio_backlog_oldest_age_seconds{queue}`: age of the oldest current pending/retry item for each bounded queue; zero when the backlog is empty.
- `synaudio_backlog_dead_letter{queue}`: current dead-letter count where the persistence model supports dead-letter state. Generation currently reports zero because terminal generation failure is represented as `FAILED`, not a dead-letter queue.

Backlog gauges are sampled from durable queue tables every 15 seconds. They are current-state gauges, not values inferred from cumulative worker counters.

## Minimum alerting/runbook

1. **Worker stale/stopped**: alert when current time minus `synaudio_worker_heartbeat_unixtime` exceeds 60 seconds for a running worker deployment. Check process/container health, DB connectivity and worker logs.
2. **Backlog age/depth**: alert when `synaudio_backlog_depth` remains non-zero while `synaudio_backlog_oldest_age_seconds` grows beyond the expected processing SLA. Split alerts by the bounded `queue` label to distinguish generation, audit and email pressure.
3. **Audit/email dead letter**: alert whenever `synaudio_backlog_dead_letter{queue="audit_outbox"}` or `{queue="email_delivery"}` is non-zero. Investigate the durable outbox before treating delivery as healthy.
4. **Retry pressure**: correlate backlog gauges with sustained growth of `audit_delivery/retrying`, stale-generation `reclaimed`, or loop `failure` counters. Inspect dependency readiness and provider logs.
5. **API 5xx trend**: compare `status_class="5xx"` request rate against total request rate per bounded route. Investigate route-specific logs and readiness dependencies.
6. **Generation/provider failures and latency**: alert on sustained increases in `synaudio_generation_jobs_total{outcome="failure"}` and on abnormal mean generation attempt duration derived from duration sum / matching outcome count. Never add raw provider error text as a metric label; use structured logs for detailed diagnosis.
7. **Readiness failure**: continue to use `/ready` for dependency gating. Metrics are diagnostic telemetry and do not replace readiness.

## Scrape examples

Prometheus should scrape API and worker metrics targets independently over their explicitly private addresses. Do not route either metrics listener through public ingress.

When adding a new worker loop, result or queue label, update the allowlist and its tests in `backend/internal/platform/metrics` in the same PR. When adding public API routes, continue using route templates rather than raw resource paths to preserve bounded cardinality.
