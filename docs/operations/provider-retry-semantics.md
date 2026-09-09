# Provider transient failure and retry semantics

This document defines the production outbound AI/TTS provider contract implemented in `backend/internal/platform/providers`.

## Retry authority layers

| Layer | Owner | Budget | When it runs |
| --- | --- | --- | --- |
| In-call provider retry | `providers` package | `1 + MaxInCallRetries` HTTP attempts per logical provider call (`MaxInCallRetries = 2`) | Inside one `generateContent` invocation for text or audio |
| Durable job retry | Generation worker (#44) | `max_attempts` on `generation_jobs` (default `3`) | After a claimed job attempt returns a `TRANSIENT` classified failure |

These layers are independent and must not duplicate authority. In-call retries recover fast transient HTTP/network failures without consuming a durable job attempt. Durable retries recover longer outages after the in-call budget is exhausted.

## Worst-case external call amplification

For a WRITER job with default settings:

```text
max durable attempts = 3
max in-call HTTP attempts per provider call = 1 + MaxInCallRetries = 3
worst-case provider HTTP calls = 3 × 3 = 9
```

TTS uses the same in-call contract at the HTTP layer. Segment orchestration is synchronous today and does not add a second hidden retry budget.

## Failure classification

| Class | Examples | In-call retry | Durable retry (#44) |
| --- | --- | --- | --- |
| `TRANSIENT` | network/timeout, HTTP 429, HTTP 5xx | Yes, bounded | Yes, while `attempt_count < max_attempts` |
| `PERMANENT` | HTTP 401/403, invalid model/config, malformed provider payload | No | No |

Stable internal codes include `PROVIDER_RATE_LIMITED`, `PROVIDER_UNAVAILABLE`, `PROVIDER_NETWORK`, `PROVIDER_AUTH`, `PROVIDER_CONFIG`, `PROVIDER_MALFORMED`, and `PROVIDER_RETRY_EXHAUSTED`. Raw provider response bodies are not exposed through public error strings.

## Backoff and throttling

- Transient in-call retries use exponential backoff (`~2s`, `~5s`, `~15s`) with deterministic jitter.
- HTTP 429 responses honor `Retry-After` when present before applying the default backoff.
- Context cancellation/deadlines stop pending backoff immediately and return the context error.

## Idempotency and provenance

Provider retries are safe retries of the same logical operation. WRITER jobs continue to:

- read frozen job input only
- reuse an existing durable output when present
- validate provenance before recording `output_ref`

Provider retries must not bypass stale-output checks or create duplicate committed revisions.

## Observability

Bounded provider metrics are exported as:

- `synaudio_provider_calls_total{provider,operation,outcome,failure_class,retry_decision}`
- `synaudio_provider_call_duration_seconds_sum{provider,operation,outcome,failure_class,retry_decision}`

Labels are allowlisted in `backend/internal/platform/metrics`. Do not add prompts, tokens, provider bodies, or generated content to logs or metrics.

## Related issues

- #44 — durable GenerationJob max-attempt authority (consumed, not redefined)
- #49 — deployment/provider degraded-state operations (consumes this runtime contract)
- #13 — user/admin failed-job recovery workflow (consumes classification)
