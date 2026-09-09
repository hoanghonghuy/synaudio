# Production deployment examples

This directory holds **provider-agnostic production examples** referenced by the canonical runbook:

- [`docs/operations/production-deployment.md`](../docs/operations/production-deployment.md)

Root [`docker-compose.yml`](../docker-compose.yml) is **development-only**. Do not deploy it to production.

## Contents

| File | Purpose |
|------|---------|
| `docker-compose.production.example.yml` | Reference topology: migration gate, API, worker, web, health probes |
| `env.production.example` | Non-secret production configuration template (secrets injected separately) |

## Usage

1. Copy and adapt the example compose file to your orchestrator (Docker Compose, Swarm, Nomad, k8s translation, etc.).
2. Inject secrets from your secret manager — never commit real values.
3. Build immutable images from a release tag:

```bash
docker build -t synaudio-api:${RELEASE} -f backend/Dockerfile backend
docker build -t synaudio-frontend:${RELEASE} -f frontend/Dockerfile frontend
```

4. Follow the ordered release sequence in the runbook: migrations → worker readiness → API `/ready` → web → smoke scripts.

## Worker probe access

Worker `/health` and `/ready` bind to `WORKER_PROBE_ADDR` (default `127.0.0.1:8081`) on a **private** address. Orchestrators must reach this bind through:

- container-native healthcheck (`curl` inside the network namespace),
- a sidecar on the same pod/network,
- or an SSH/tunnel from the operator host for manual smoke.

Do **not** expose worker probes on public ingress.
