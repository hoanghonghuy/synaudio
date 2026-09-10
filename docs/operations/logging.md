# Application logging privacy and correlation

Synaudio application logs use a repository-owned redaction and correlation contract. This policy is distinct from, but compatible with, durable audit redaction (#10) and metrics privacy (#37).

## What must never appear in application logs

- `Authorization` / `Cookie` headers and raw bearer tokens
- Access, refresh, reset, verification, and MFA secrets or recovery codes
- API keys, provider secrets, and `DATABASE_URL` credentials
- Presigned object URLs and transactional-email action links
- Full prompts, story/chapter text, generated audio bytes, or raw provider response bodies

Audit events remain the durable semantic record for security-sensitive mutations. Metrics remain bounded operational telemetry. Application logs provide incident diagnosis with safe fields only.

## Safe logger contract

- API and worker binaries construct loggers via `backend/internal/platform/logging.New`.
- `LOG_LEVEL` accepts `debug`, `info`, `warn`, or `error`. Production defaults to `info`; verbose payload logging is not a supported troubleshooting path.
- All structured attributes pass through the redacting handler. Sensitive attribute keys are replaced with `[REDACTED]`, including entire structured groups whose parent key is sensitive (generic child field names do not bypass the policy). Free-form log messages and string values are scanned for credential, presigned URL, and action-link patterns.
- Boundary code must log errors with `logging.ErrAttr(err)` or `logging.SafeFailureFields(err)` instead of raw `error` values.

## Request correlation (API)

- Chi generates a bounded `request_id` for every request.
- Clients may send `X-Correlation-ID` using `[A-Za-z0-9_-]` up to 128 characters. Invalid or oversized values are ignored and replaced with the generated `request_id`.
- `WithRequestLogger` emits one bounded request record per request with `method`, `route` (chi route pattern), `status`, `latency_ms`, `request_id`, and `correlation_id`.
- Recovered panics emit `request panic recovered` with the same correlation fields and HTTP 500 semantics. Panic values, headers, and bodies are never logged.

## Worker / provider correlation

- Worker logs include bounded durable identifiers such as `worker_id`, `job_id`, and `run_id`.
- Generation failures should log `error_class` and `error_code` from classified errors instead of raw upstream text.
- Provider and storage boundaries must map failures to safe summaries before they reach composition-root logging.

## Operations workflow

1. Start from `request_id` / `correlation_id` in API logs, then pivot to audit queries by `correlation_id` when a security-sensitive mutation is involved.
2. For generation incidents, correlate worker `job_id` / `run_id` with audit `generation_run_id` and bounded metrics (`error_class`, `job_type`).
3. Use structured application logs for provider failure detail; never add raw provider messages as metric labels (#37).
4. When adding new logging call sites, use the safe logger helpers and extend redaction regression tests in the same PR.

## Related runbooks

- Metrics privacy and alerting: `docs/operations/observability.md`
- Worker identity: `docs/operations/worker-identity.md`
- Audit semantics and durable provenance: audit package and #10 implementation
