# AI Audiobook Platform (Synaudio)

Monorepo triển khai theo `docs/ai-audiobook-spec/spec_final.md`,
`docs/ai-audiobook-spec/SPEC-AMENDMENT-001-POST-VERIFICATION.md`, và
`docs/ai-audiobook-spec/SPEC-AMENDMENT-002-SAME-ORIGIN-PRODUCTION.md`.

## Current phase

**Phase 0 — Foundation**

## Layout

```text
backend/     Go API + Worker (modular monolith)
frontend/    Vue 3 + Vite + TypeScript
docs/        Final specification + historical docs
docker-compose.yml
```

## Quick start

**Local development only.** For production rollout, use [`docs/operations/production-deployment.md`](docs/operations/production-deployment.md) and [`deploy/`](deploy/). Root `docker-compose.yml` is not a production deployment authority.

1. Copy environment template:

```bash
cp .env.example .env
```

2. Start local infrastructure and apply database migrations:

```bash
docker compose up -d postgres minio minio-init migrate
```

The API/worker expect the schema to be migrated before startup. The compose `backend-api` and `backend-worker` services already depend on the `migrate` service completing successfully.

3. Run backend tests:

```bash
cd backend && go test ./...
```

4. Run the repository-owned Chapter → Audio → Listen smoke proof (#97):

```bash
make smoke
```

This deterministic smoke uses in-memory stores and mock AI/TTS providers. It traverses real HTTP handlers and domain services (generation, approval, narration, synthesize, activate, publish, listener audio URL) without manual database edits.

5. Run API / Worker:

```bash
cd backend
go run ./cmd/api
go run ./cmd/worker
```

6. Run frontend:

```bash
cd frontend
npm install
npm run dev
```

## Runtime AI/TTS providers

Provider selection is controlled by `AI_MODE` and `TTS_MODE`.

- Local development and deterministic tests may explicitly use `mock`.
- Gemini mode uses `GEMINI_API_KEY`, `GEMINI_TEXT_MODEL`, `GEMINI_TTS_MODEL`, and `GEMINI_TTS_VOICE` from the environment.
- Production rejects mock AI/TTS configuration and does not silently fall back when a real provider is unsupported or misconfigured.
- `GEMINI_TTS_VOICE` is a Gemini provider-native voice name. Synaudio's logical narration `voice_id` remains a separate application identity and is not sent to Gemini as `voiceName`.

Use `.env.example` as the canonical runtime environment template.

## Production browser/API authority

Synaudio V1 supports one canonical browser origin in production:

```text
Browser
  -> https://app.example.com
      -> frontend SPA
      -> relative /api/v1/*
          -> private backend API
```

Operators must terminate public HTTPS at the trusted web/ingress layer and keep the backend API private behind that same-origin proxy. Browser code must continue to call relative `/api/v1`; do not expose a second public browser API origin or add permissive credentialed CORS as a deployment shortcut.

Cookie-backed refresh/logout remains protected by the repository Origin/CSRF policy even though the canonical browser path is same-origin. `Forwarded` / `X-Forwarded-*` values are trusted only when the immediate peer is configured in `TRUSTED_PROXY_CIDRS`; direct clients must not be able to spoof forwarded HTTPS/host identity. Bearer-authenticated non-browser API clients remain supported independently of browser CORS.

See [`SPEC-AMENDMENT-002-SAME-ORIGIN-PRODUCTION.md`](docs/ai-audiobook-spec/SPEC-AMENDMENT-002-SAME-ORIGIN-PRODUCTION.md) and [`docs/operations/production-deployment.md`](docs/operations/production-deployment.md) for the authoritative contract and rollout checks.

## Health endpoints

**API (public listener)**

- `GET /health` — process liveness, no dependency calls.
- `GET /ready` — readiness across configured critical dependencies. Current API composition includes database and object-storage checks, plus FFmpeg when the production FFmpeg processor is enabled; any failing dependency returns a non-ready response with per-dependency status.

**Worker (private probe on `WORKER_PROBE_ADDR`, default `127.0.0.1:8081`)**

- `GET /health` — worker process liveness.
- `GET /ready` — PostgreSQL connectivity and main-loop heartbeat freshness (≤60s). Does **not** cover API readiness and does not poll live AI/TTS providers. See [`docs/operations/production-deployment.md`](docs/operations/production-deployment.md).

## Production deployment

Canonical release/runbook: [`docs/operations/production-deployment.md`](docs/operations/production-deployment.md)

Provider-agnostic examples: [`deploy/`](deploy/)

## Spec precedence

1. `docs/ai-audiobook-spec/spec_final.md`
2. `docs/ai-audiobook-spec/SPEC-AMENDMENT-001-POST-VERIFICATION.md`
3. `docs/ai-audiobook-spec/SPEC-AMENDMENT-002-SAME-ORIGIN-PRODUCTION.md` — supersedes Amendment 001 browser-origin/CORS wording where they conflict.
