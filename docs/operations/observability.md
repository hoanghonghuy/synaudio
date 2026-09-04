# Production observability

Synaudio exposes a Prometheus-compatible metrics surface without replacing the existing health/readiness contract.

## Endpoints

- API: `GET /metrics` on the normal API listener.
- Worker: `GET /` on `WORKER_METRICS_ADDR` (default `:9091`). Set this address explicitly in production when the worker metrics listener must bind to a different interface/port.

The metrics listener must be restricted to the private monitoring network or protected by infrastructure controls. It is not a public product endpoint.

## Cardinality and privacy

Metrics use bounded labels only. Do not add user IDs, story/chapter/job IDs, raw URLs, query strings, prompts, generated content, email addresses, auth/recovery tokens, provider error messages, or action links as labels.

API request metrics use the chi route pattern after routing, not the raw request path. Unknown/invalid routes collapse to `unmatched`. Worker loop, result, job-type and error-class labels are explicit allowlists and collapse unknown values to `other`/`UNKNOWN`.

## Metrics

- `synaudio_api_requests_total{method,route,status_class}`: API traffic and status-class trend.
- `synaudio_api_request_duration_seconds_sum{method,route,status_class}`: cumulative request latency. Divide by the matching request count for mean latency.
- `synaudio_worker_heartbeat_unixtime`: last observed worker loop heartbeat.
- `synaudio_worker_loop_runs_total{loop,outcome}`: success/failure of generation polling, stale reclaim, audit delivery, transactional email delivery and account-deletion reconciliation.
- `synaudio_worker_loop_items_total{loop,result}`: bounded item outcomes including reclaimed, processed, claimed, delivered, retrying, dead-letter and purged.
- `synaudio_generation_jobs_total{job_type,outcome,error_class}`: bounded generation outcome/error-class signal.

## Minimum alerting/runbook

1. **Worker stale/stopped**: alert when current time minus `synaudio_worker_heartbeat_unixtime` exceeds 60 seconds for a running worker deployment. Check process/container health, DB connectivity and worker logs.
2. **Audit dead letter**: alert on any increase of `synaudio_worker_loop_items_total{loop="audit_delivery",result="dead_letter"}`. Investigate the durable audit outbox before treating audit delivery as healthy.
3. **Retry pressure**: alert on sustained growth of `audit_delivery/retrying`, stale-generation `reclaimed`, or loop `failure` counters. Correlate with dependency readiness and provider logs.
4. **API 5xx trend**: compare `status_class="5xx"` request rate against total request rate per bounded route. Investigate route-specific logs and readiness dependencies.
5. **Generation/provider failures**: alert on sustained increases in `synaudio_generation_jobs_total{outcome="failure"}` by bounded job/error class. Never add the raw provider error as a metric label; use structured logs for detailed diagnosis.
6. **Readiness failure**: continue to use `/ready` for dependency gating. Metrics are diagnostic telemetry and do not replace readiness.

## Scrape examples

Prometheus should scrape the API and worker targets independently. A minimal configuration can use the service DNS names and private ports for each deployment. Keep the metrics endpoints off the public ingress where possible.

When adding a new worker loop or result label, update the allowlist and its tests in `backend/internal/platform/metrics` in the same PR. When adding public API routes, continue using route templates rather than raw resource paths to preserve bounded cardinality.
