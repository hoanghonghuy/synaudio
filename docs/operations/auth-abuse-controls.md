# Auth abuse controls

Production API instances enforce bounded request behavior on sensitive authentication, recovery, and verification routes before handler logic runs.

## Runtime boundary

- Middleware is mounted on `/api/v1/auth` for login, registration, refresh, email verification/resend, password forgot/reset, MFA challenge/verification surfaces, and privileged `POST /re-auth` MFA verification.
- `/health`, `/ready`, and ordinary authenticated API traffic outside `/api/v1/auth` are not covered by this middleware.
- Development defaults to an in-memory limiter (`AUTH_ABUSE_BACKEND=memory`). Production requires shared PostgreSQL counters (`AUTH_ABUSE_BACKEND=postgres`) so limits are not multiplied per replica.

## Trusted client identity

Configure `TRUSTED_PROXY_CIDRS` with the CIDR blocks of the canonical reverse proxy or ingress that terminates public traffic.

- When unset, the API ignores `X-Forwarded-For` and `X-Real-IP` and uses the direct TCP peer (`RemoteAddr`) for client-scoped limits and session IP metadata.
- When set, forwarded headers are honored only when the immediate peer address is inside one of the configured CIDRs.

The bundled frontend nginx proxy sets `X-Real-IP`, `X-Forwarded-For`, and `X-Forwarded-Proto`. Production ingress for issue #49 must preserve the same trustworthy boundary or provide an explicitly authoritative ingress rate limit that consumes the policy documented here.

## Throttle contract

- Non-enumeration routes return `429` with `{"error":{"code":"RATE_LIMITED","message":"too many requests"}}` and a `Retry-After` header in seconds.
- Password forgot and email resend remain enumeration resistant: throttled requests still return `202 {"status":"accepted"}`.
- Metrics: `synaudio_auth_throttled_total{route,dimension}` with bounded `route` and `dimension` labels (`client`, `account`, `session`).
- Structured logs emit `auth request throttled` with masked client IP prefix and request ID only.

## Deployment dependency for #49

Multi-instance production must use `AUTH_ABUSE_BACKEND=postgres` with migration `000018_auth_abuse_counters` applied, or an ingress control that enforces the same per-route budgets authoritatively. Per-replica in-memory limits are not a production-safe control.
