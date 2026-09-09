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

echo "==> Format-check changed Go files"
if git remote get-url origin >/dev/null 2>&1; then
  git fetch -q origin develop
fi
base_ref="${PRE_PUSH_BASE_REF:-origin/develop}"
if git rev-parse --verify "$base_ref" >/dev/null 2>&1; then
  base_sha="$(git merge-base HEAD "$base_ref")"
else
  base_sha="$(git rev-parse HEAD^)"
fi
mapfile -t go_files < <(
  {
    git diff --name-only --diff-filter=ACMRT "$base_sha"...HEAD
    git diff --name-only --diff-filter=ACMRT
    git diff --cached --name-only --diff-filter=ACMRT
  } | grep '^backend/.*\.go$' | sort -u
)
if [ "${#go_files[@]}" -gt 0 ]; then
  unformatted="$(gofmt -l "${go_files[@]}")"
  if [ -n "$unformatted" ]; then
    echo "Changed Go files requiring gofmt:" >&2
    echo "$unformatted" >&2
    exit 1
  fi
fi

echo "==> Dependency supply-chain gates"
bash ./scripts/govulncheck-gate.sh
bash ./scripts/test-govulncheck-gate.sh

echo "==> Backend checks"
cd backend
go test ./...
go vet ./...
go build -o bin/api ./cmd/api
go build -o bin/worker ./cmd/worker

cd "$ROOT"
echo "==> Frontend checks"
cd frontend
npm ci --prefer-offline --no-fund
cd "$ROOT"
bash ./scripts/npm-audit-gate.sh
bash ./scripts/test-npm-audit-gate.sh
cd frontend
npm test
npm run typecheck
npm run build

echo "Pre-push verification passed."
