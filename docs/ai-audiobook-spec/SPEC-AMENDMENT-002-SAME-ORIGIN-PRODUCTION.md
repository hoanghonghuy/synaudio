# SPEC AMENDMENT 002 — Same-Origin Production Browser Authority

**Status:** REQUIRED FOR IMPLEMENTATION  
**Amendment ID:** `SPEC-AMENDMENT-002`  
**Decision source:** GitHub #62  
**Supersedes:** the browser-origin/CORS topology wording in `SPEC-AMENDMENT-001` A-001 where the two conflict.

## Final V1 production decision

Synaudio V1 uses one public browser origin behind a reverse proxy:

```text
Browser
  -> public HTTPS web origin
      -> frontend SPA
      -> relative /api/v1/* proxy
          -> private backend API
```

The browser MUST NOT require credentialed cross-origin API access in the canonical V1 deployment. Frontend API calls therefore remain relative (`/api/v1`), and production configuration MUST NOT advertise a generic `CORS_ALLOWED_ORIGINS` switch as if a second public API origin were supported.

`APP_PUBLIC_URL` and `API_PUBLIC_URL`, when required by non-browser server concerns, MUST describe the deployed same-origin authority and MUST NOT be interpreted as permission to expose a second browser API origin.

## Authentication and CSRF boundary

Same-origin deployment removes the need for browser CORS; it does **not** remove CSRF protections for cookie-backed state changes.

- Refresh/session cookies remain `HttpOnly`, `Secure` in production, host-only, and `SameSite=Lax` unless a later reviewed amendment changes that contract.
- Cookie-authenticated state-changing requests MUST enforce the repository-owned Origin/CSRF policy at the trusted application/ingress boundary.
- Bearer-authenticated non-browser API clients are not governed by browser CORS and MUST remain supported.
- Forwarded scheme/host information is authoritative only from explicitly trusted reverse proxies; an arbitrary direct client MUST NOT be able to spoof `X-Forwarded-*` values to upgrade cookie/security trust.

## Configuration consequence

The following browser-topology configuration is obsolete for V1 and must be removed from runtime/examples:

```text
CORS_ALLOWED_ORIGINS
```

Do not add permissive wildcard CORS as a compatibility fallback. A future cross-origin browser mode requires a new explicit product/security decision, exact-origin credential-aware policy, and regression coverage.

## Verification

Implementation consuming this amendment must prove:

1. production and local frontend paths continue to use relative `/api/v1`;
2. no dead/misleading browser CORS knob remains;
3. cookie-backed refresh/logout or equivalent state changes fail closed under the selected Origin/CSRF policy;
4. trusted-proxy handling cannot be upgraded by arbitrary forwarded headers;
5. existing health/readiness/private operator and non-browser Bearer paths remain unaffected.
