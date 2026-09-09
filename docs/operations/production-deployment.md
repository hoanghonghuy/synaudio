# Production release and deployment contract

This document is the **canonical production rollout authority** for Synaudio. It is distinct from the local development quick-start in the repository root `README.md` and from root `docker-compose.yml`, which is explicitly **development-only** (`APP_ENV=development`, mock AI/TTS, local Postgres/MinIO credentials, published DB/MinIO ports).

Provider-agnostic examples live under `deploy/`. Same-origin browser ingress semantics are owned by #48; this runbook consumes that contract without redefining CORS/origin design.

## Production components and topology

| Component | Role | Machine-checkable health |
|-----------|------|--------------------------|
| Web / ingress | Serves the Vue SPA and reverse-proxies `/api`, `/health`, `/ready` to the API | HTTP 200 on static shell + proxied API liveness |
| API | Public HTTP product surface (`HTTP_ADDR`, default `:8080`) | `GET /health` (liveness), `GET /ready` (dependency readiness) |
| Worker | Background generation, audit/email delivery, account deletion | `GET /health` and `GET /ready` on `WORKER_PROBE_ADDR` (default `127.0.0.1:8081`) |
| PostgreSQL | Authoritative relational state | Managed service health (operator responsibility) |
| Object storage | Private media objects (S3-compatible, TLS) | API `/ready` storage check + operator bucket-policy verification |
| Migration job | One-shot schema apply before app promotion | Exit code `0`; failure **blocks** API/worker rollout |
| AI / TTS providers | External generation | Config validated at startup; transient outage is **degraded work**, not liveness failure |
| Metrics (optional) | Private Prometheus scrape targets | Not a promotion gate; diagnostic only |

Typical topology:

```text
Browser ──► HTTPS ingress (web + /api proxy)
              ├──► frontend container (unprivileged nginx :8080)
              └──► API container (:8080)
Worker container(s) ──► PostgreSQL
                   └──► object storage (TLS)
                   └──► private probe/metrics (loopback/private bind only)
Migration job (one-shot) ──► PostgreSQL
```

## Development vs production separation

Production **must not** silently run with:

- `APP_ENV=development`
- `AI_MODE=mock` / `TTS_MODE=mock` / `AUDIO_PROCESSOR_MODE=mock`
- `STORAGE_PROVIDER=minio` (local-only provider)
- `ALLOW_REMOTE_DATABASE_IN_DEV=true` or `ALLOW_REMOTE_STORAGE_IN_DEV=true`
- Default credentials from root `docker-compose.yml`

Application startup enforces these rules in `backend/internal/platform/config`. Release verification must include a config/startup check proving the deployed artifact runs under `APP_ENV=production` with real providers (#59).

## Immutable release artifacts

Build and promote **identifiable immutable artifacts**:

1. **API/worker image** — `backend/Dockerfile` produces `/app/api` and `/app/worker` on pinned `alpine:3.21.3`, running as UID/GID `65532`.
2. **Frontend image** — `frontend/Dockerfile` produces static assets on pinned `nginxinc/nginx-unprivileged:1.27.4-alpine3.21`, listening on `:8080` without root.
3. **Migration inputs** — SQL files in `backend/db/migrations/` at the same Git tag/commit as the application images.

Tag every release with a unique version (Git SHA or semver). Do **not** deploy floating `latest` tags in production. Record the promoted digest/tag in change-management notes.

Build verification is part of CI (`go build` API/worker, `npm run build` frontend). Release evidence must also include green dependency/supply-chain gates from `docs/operations/dependency-supply-chain-security.md` (#53) when that document is present in the branch.

## Configuration and secrets injection

- Inject secrets via the orchestrator/secret manager. **Never** commit production credentials, API keys, or token signing keys into images or repository-owned production manifests.
- Required production configuration is documented in `.env.example`. Minimum set:
  - `APP_ENV=production`
  - `DATABASE_URL` (TLS)
  - `STORAGE_PROVIDER=r2` (or other supported S3-compatible provider), `STORAGE_ENDPOINT` (**HTTPS**), bucket and least-privilege keys
  - Real AI/TTS provider settings (`AI_MODE`, `TTS_MODE`, provider keys/models)
  - `ACCESS_TOKEN_ACTIVE_KID` / `ACCESS_TOKEN_KEYS` keyring
  - `EMAIL_MODE=smtp` and SMTP credentials when email is enabled
  - Distinct `WORKER_ID` per concurrent worker replica (`docs/operations/worker-identity.md`)
- Optional private telemetry binds: `API_METRICS_ADDR`, `WORKER_METRICS_ADDR`, `WORKER_PROBE_ADDR` (loopback/private IP only).

When #50 auth-abuse controls are deployed, production requires `AUTH_ABUSE_BACKEND=postgres` and migration `000018` before API rollout (`docs/operations/auth-abuse-controls.md`).

## Object storage security contract

Synaudio presigns private object downloads through application eligibility gates (#13). **Bucket privacy is operator-enforced**; the application does not configure provider IAM/bucket policy.

Production operators **must verify**:

1. **TLS** — `STORAGE_ENDPOINT` uses `https://` (enforced at startup).
2. **No anonymous reads** — bucket/object policy denies unauthenticated `GetObject` / public ACLs.
3. **Least privilege** — application credentials can `PutObject`, `GetObject`, `DeleteObject`, `HeadObject` on the media prefix only; no account-wide admin keys in the runtime.
4. **Authorized client access only** — listeners receive time-limited presigned URLs from the API after eligibility checks; never expose the bucket as a public CDN substitute.

Post-deploy verification **must** include:

- **Negative check** — unauthenticated direct object URL returns `403`/`404` (not `200`). Use `scripts/verify-storage-privacy.sh`.
- **Positive check (when eligible)** — authenticated listener flow returns a presigned URL that succeeds for an entitled object (product smoke; no uncontrolled AI/TTS generation).

If internal `STORAGE_ENDPOINT` differs from the client-reachable host used in presigned URLs, configure the provider/signing endpoint so browser fetches succeed over HTTPS.

## Schema migrations (gated release step)

1. Run a **one-shot migration job** against production PostgreSQL **before** promoting API/worker images that depend on the new schema.
2. Use the same migration tool/version as CI/dev: `migrate/migrate:v4.18.3` with `-path=/migrations` and production `DATABASE_URL`.
3. **Migration failure stops rollout** — do not start or promote API/worker until migrations succeed.
4. Keep migration job logs as release evidence.

### Forward-compatible migration / rollback policy

- Prefer **expand/contract** migrations compatible with rolling deploys.
- Application rollback = redeploy previous **known-good image tag**; do **not** run destructive `down` migrations as the normal rollback mechanism.
- If a migration is expand-only, older app versions may continue running during rollout; contract/drop phases require all replicas on the new version before applying destructive steps.

## Release sequence (ordered)

1. **Pre-flight** — CI green on the release commit (#53 supply-chain gates when available); change window approved.
2. **Build & publish** — immutable API/worker/frontend artifacts tagged with the release ID.
3. **Configure secrets** — production env validated offline against `.env.example` rules.
4. **Verify storage policy** — bucket privacy checklist complete (see above).
5. **Run migrations** — one-shot job; abort on non-zero exit.
6. **Roll out worker(s)** — start new replicas; wait until each passes worker readiness (below). Ensure unique `WORKER_ID` per replica.
7. **Roll out API** — start/promote only after `/ready` succeeds (database, storage, FFmpeg when enabled).
8. **Roll out web/ingress** — deploy frontend + ingress aligned with #48 same-origin contract and timeout alignment below.
9. **Post-deploy smoke** — `scripts/verify-production-smoke.sh` (no live AI/TTS generation).
10. **Observe** — metrics/alerts per `docs/operations/observability.md`.

## API readiness and liveness

Reuse existing endpoints on the public API listener:

| Probe | Path | Pass | Fail |
|-------|------|------|------|
| Liveness | `GET /health` | `200 {"status":"ok"}` | non-200 / connection failure |
| Readiness | `GET /ready` | `200` with `status=ready` and required dependencies `ok` | `503` with `degraded` / `unavailable` |

Current dependency composition (`backend/cmd/api/main.go`):

- `database` — PostgreSQL ping
- `storage` — object storage ping
- `ffmpeg` — when production FFmpeg processor is enabled

**Do not** add live AI/TTS provider polling to `/ready`. Invalid provider **configuration** fails fast at startup; transient provider outage affects job processing metrics and retry semantics (#55) without killing API liveness.

Graceful shutdown: API handles `SIGINT`/`SIGTERM` with a **10s** `Shutdown` window (`backend/cmd/api/main.go`).

## Worker health and readiness (separate contract)

**API `/ready` does not cover the worker process.** Worker rollout must use the worker probe server:

| Probe | Path (on `WORKER_PROBE_ADDR`) | Pass | Fail |
|-------|-------------------------------|------|------|
| Liveness | `GET /health` | `200 {"status":"ok"}` | non-200 |
| Readiness | `GET /ready` | `200 {"status":"ready"}` with `database=ok`, `worker_loop=ok` | `503` draining/degraded |

Readiness semantics (`backend/internal/platform/workerprobe`):

- **`database`** — PostgreSQL ping (2s timeout). Failure → `degraded`, worker not promoted.
- **`worker_loop`** — main loop heartbeat freshness ≤ **60s** (`synaudio_worker_heartbeat_unixtime` / `Registry.HeartbeatAge`). Stale heartbeat indicates dead/stuck worker → not ready.
- **`draining`** — after `SIGINT`/`SIGTERM`, readiness returns `503` with `status=draining` so orchestrators stop sending traffic before process exit.

External AI/TTS providers are **excluded** from worker readiness. Provider quota/outage surfaces through generation job failure metrics and existing retry authority; the worker remains ready if it can reach PostgreSQL and its loop is alive.

Authoritative heartbeat: updated on generation poll, stale reclaim, audit delivery, email delivery, and account-deletion loops (`backend/cmd/worker/main.go`). Alert when `now - synaudio_worker_heartbeat_unixtime > 60` (`docs/operations/observability.md`).

Private metrics on `WORKER_METRICS_ADDR` remain diagnostic; probes for promotion use `/health` and `/ready`.

### Worker shutdown and lease recovery

On termination:

1. Readiness flips to **draining** immediately.
2. In-flight job cancellation leaves durable leases **RUNNING** for stale reclaim (#60/#76); do not treat signal cancellation as a permanent job failure.
3. Set orchestrator **termination grace ≥ 5 minutes** (current SQL lease horizon) so reclaimed work is not duplicated beyond existing lease/retry semantics.
4. Old and new worker replicas during rolling updates must use **different** `WORKER_ID` values.

## Web / ingress timeout alignment (#52)

Application HTTP bounds are defined in `docs/operations/observability.md` and `backend/internal/platform/httpserver`:

- Public API `WriteTimeout`: **11m** (longest synchronous admin generation path)
- Public API `ReadTimeout`: **60s**; body limit **2 MiB**
- Metrics `WriteTimeout`: **30s**

Ingress/reverse-proxy timeouts for proxied API routes must be **≥ application bounds** (see `frontend/nginx.conf` production example: `proxy_read_timeout` / `proxy_send_timeout` 660s for `/api`). Health/ready routes use shorter 15s proxy timeouts.

Do not configure ingress body limits **looser** than the application router limit in a way that makes the app an unbounded fallback.

## PostgreSQL connection budget (#54)

Each API and worker replica creates an independent `pgxpool` with library-default `MaxConns` until #54 publishes explicit pool caps. When sizing replicas and rolling-update surge:

```text
estimated_connections =
  (api_replicas + worker_replicas + surge_old_new_overlap) × per_process_default_max
  + migration_job_connections
  + operational_reserve
```

Do not multiply replicas without checking the managed PostgreSQL connection limit. Consume #54's declared budget when it lands; until then, treat simultaneous old+new replicas during rolling updates as requiring **2×** per-role connection headroom.

## External AI/TTS degraded-state semantics

| Condition | Behavior |
|-----------|----------|
| Invalid/missing production provider config | **Fail fast** at process startup (API and worker) |
| Transient provider 429/5xx/network errors | Jobs classified per #55/#44 retry authority; observable via `synaudio_generation_jobs_total` and structured logs |
| Worker readiness | **Unaffected** by transient provider outage |
| API `/ready` | **Must not** poll live providers |
| Post-deploy smoke | **Must not** trigger uncontrolled live generation |

## Rollback and abort procedures

### Migration failure

1. **Stop** — do not deploy API/worker/web.
2. **Investigate** migration logs; fix forward-only migration or restore DB from backup (#51) if needed.
3. **Do not** run destructive down migrations as the default recovery.

### Application rollout failure (after successful migration)

1. **Stop promotion** — mark new replicas not ready / remove from load balancing.
2. **Roll back images** to the previous known-good tag for API, worker, and frontend.
3. **Verify** `/ready`, worker `/ready`, and smoke script on rolled-back version.
4. Forward-only schema must remain compatible with the rolled-back app version.

### Abort mid-rollout

- Scale down new replicas failing readiness.
- Keep database and storage unchanged unless migration succeeded only partially (treat as migration failure).

## Post-deploy smoke verification

Run from an operator workstation with network access to production endpoints:

```bash
./scripts/verify-production-smoke.sh \
  --api-base https://app.example.com \
  --worker-probe http://127.0.0.1:8081

./scripts/verify-storage-privacy.sh \
  --unsigned-url 'https://media.example.com/bucket/object-key'
```

Smoke checks (no live AI/TTS calls):

1. `GET /health` → 200
2. `GET /ready` → 200 with required dependencies ok
3. Worker `GET /health` and `GET /ready` → 200 (via sidecar/SSH tunnel to private probe)
4. Web shell loads (HTTP 200 on `/`)
5. Storage negative access denied (unsigned object URL)
6. Optional: authenticated non-generation API path (e.g. catalog health) when credentials available

## Container privilege and filesystem

Production containers must satisfy:

- API/worker run as **non-root** UID `65532` (image default).
- Frontend runs as nginx-unprivileged user on port **8080**.
- No privileged mode, host networking, or Docker socket mounts for normal operation.
- Prefer read-only root filesystem with explicit writable temp mounts for worker media staging (#58).

Release smoke must prove processes start and pass health/readiness under this policy.

## Related operational docs

- `docs/operations/observability.md` — metrics, alerts, heartbeat threshold
- `docs/operations/worker-identity.md` — `WORKER_ID` requirements
- `docs/operations/access-token-key-rotation.md` — signing key rollout
- `docs/operations/dependency-supply-chain-security.md` — #53 release evidence (#53)
- `deploy/README.md` — compose/orchestrator examples
