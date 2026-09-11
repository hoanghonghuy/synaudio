# Production deployment examples

This directory holds **provider-agnostic production examples** referenced by the canonical runbook:

- [`docs/operations/production-deployment.md`](../docs/operations/production-deployment.md)
- [`docs/ai-audiobook-spec/SPEC-AMENDMENT-002-SAME-ORIGIN-PRODUCTION.md`](../docs/ai-audiobook-spec/SPEC-AMENDMENT-002-SAME-ORIGIN-PRODUCTION.md)

Root [`docker-compose.yml`](../docker-compose.yml) is **development-only**. Do not deploy it to production.

## Canonical browser topology

V1 has one public browser origin. The deployment edge serves the SPA and proxies relative `/api/v1/*` traffic to a private backend API:

```text
Browser
  -> public HTTPS web/ingress
      -> frontend SPA
      -> /api/v1/* -> private backend API
```

Do not publish a separate browser-facing API origin or add permissive credentialed CORS to these examples. If `APP_PUBLIC_URL` and `API_PUBLIC_URL` are both configured, they must describe the same browser authority for V1.

The immediate reverse proxy must be the only source of trusted forwarded scheme/host data. Configure its network in `TRUSTED_PROXY_CIDRS`; direct clients must not be able to make the API trust spoofed `Forwarded` or `X-Forwarded-*` headers. Cookie-backed refresh/logout remains subject to the application Origin/CSRF checks. Bearer-authenticated non-browser clients remain independent of browser CORS.

## Contents

| File | Purpose |
|------|---------|
| `docker-compose.production.example.yml` | Reference topology: migration gate, API, worker, web, health probes |
| `env.production.example` | Non-secret production configuration template (secrets injected separately) |

## Usage

1. Copy and adapt the example compose file to your orchestrator (Docker Compose, Swarm, Nomad, k8s translation, etc.).
2. Inject secrets from your secret manager — never commit real values.
3. Terminate public HTTPS at the trusted web/ingress layer and keep the backend API private behind the same-origin proxy.
4. Build immutable images from a release tag:

```bash
docker build -t synaudio-api:${RELEASE} -f backend/Dockerfile backend
docker build -t synaudio-frontend:${RELEASE} -f frontend/Dockerfile frontend
```

5. Follow the ordered release sequence in the runbook: migrations → worker readiness → API `/ready` → web → smoke scripts.

## Worker probe access

Worker `/health` and `/ready` bind to `WORKER_PROBE_ADDR` (default `127.0.0.1:8081`) on a **private** address. Orchestrators must reach this bind through:

- container-native healthcheck (`curl` inside the network namespace),
- a sidecar on the same pod/network,
- or an SSH/tunnel from the operator host for manual smoke.

Do **not** expose worker probes on public ingress.
