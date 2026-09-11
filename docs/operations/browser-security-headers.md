# Production browser security headers

This document defines the repository-owned browser response-hardening baseline for Synaudio V1. It consumes the single-public-origin decision in `SPEC-AMENDMENT-002-SAME-ORIGIN-PRODUCTION.md` and does not redefine the ingress topology.

## Web-container policy

`frontend/nginx.conf` serves the production SPA and emits the baseline headers on browser-facing responses:

- `Content-Security-Policy`
  - `default-src 'self'`
  - scripts, styles, API connections, forms, and base URLs are same-origin
  - `object-src 'none'` and `frame-ancestors 'none'`
  - `media-src 'self' https: blob:` intentionally permits HTTPS presigned private-media URLs because the configured object-storage delivery hostname may differ from the application origin
  - no `unsafe-eval`, `unsafe-inline`, or wildcard source fallback
- `X-Frame-Options: DENY` as legacy clickjacking defense alongside CSP `frame-ancestors`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy` disables camera, microphone, geolocation, payment, USB, accelerometer, gyroscope, and magnetometer because V1 does not require those browser capabilities

Local Vite development is intentionally not forced through this production-only Nginx policy.

## HSTS ownership

The frontend container listens on plain HTTP `:8080` behind the production HTTPS ingress. It therefore MUST NOT emit `Strict-Transport-Security`: an HTTP-only internal hop cannot authoritatively assert transport security.

The public TLS terminator / ingress owns HSTS. Production ingress should emit `Strict-Transport-Security` only on HTTPS responses after TLS is successfully terminated. A recommended baseline is `max-age=31536000; includeSubDomains` only when the operator has verified that all relevant subdomains are HTTPS-only; `preload` requires a separate deliberate operational decision.

Do not enable HSTS on local development or any public endpoint that can still be served over plaintext HTTP.

## Verification

`frontend/test/production-security-headers.test.mjs` is a deterministic policy regression test. It fails if required headers disappear, CSP is broadened with wildcard/unsafe script execution, or HSTS is moved into the HTTP-only frontend container.

Production rollout should additionally verify the actual public HTTPS ingress response with a smoke request so that edge-owned HSTS and web-container-owned headers are both present after proxying.
