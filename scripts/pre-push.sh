#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

export TEST_DATABASE_URL="${TEST_DATABASE_URL:-postgres://synaudio:synaudio@localhost:5432/synaudio?sslmode=disable}"

echo "==> Start local integration services"
docker compose config >/dev/null
docker compose up -d --wait postgres minio
docker compose run --rm minio-init >/dev/null

echo "==> Verify generated SQL code"
make sqlc-check
make sqlc-check-regression

echo "==> Backend checks"
cd backend
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
  echo "Unformatted Go files:" >&2
  echo "$unformatted" >&2
  exit 1
fi
go test ./...
go vet ./...
go build -o bin/api ./cmd/api
go build -o bin/worker ./cmd/worker

cd "$ROOT"
echo "==> Frontend checks"
cd frontend
npm ci --prefer-offline --no-audit --no-fund
npm test
npm run typecheck
npm run build

echo "Pre-push verification passed."
