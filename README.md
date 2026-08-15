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

2. Start local infrastructure:

```bash
docker compose up -d postgres minio minio-init
```

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

## Health endpoints

- `GET /health` — process liveness, no dependency calls
- `GET /ready` — database readiness

## Spec precedence

1. `docs/ai-audiobook-spec/spec_final.md`
2. `docs/ai-audiobook-spec/SPEC-AMENDMENT-001-POST-VERIFICATION.md`
