# AI Audiobook Platform (Synaudio)

Monorepo triển khai theo `docs/ai-audiobook-spec/spec_final.md` và
`docs/ai-audiobook-spec/SPEC-AMENDMENT-001-POST-VERIFICATION.md`.

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

4. Run API / Worker:

```bash
cd backend
go run ./cmd/api
go run ./cmd/worker
```

5. Run frontend:

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

## Health endpoints

- `GET /health` — process liveness, no dependency calls.
- `GET /ready` — readiness across configured critical dependencies. Current API composition includes database and object-storage checks, plus FFmpeg when the production FFmpeg processor is enabled; any failing dependency returns a non-ready response with per-dependency status.

## Spec precedence

1. `docs/ai-audiobook-spec/spec_final.md`
2. `docs/ai-audiobook-spec/SPEC-AMENDMENT-001-POST-VERIFICATION.md`
