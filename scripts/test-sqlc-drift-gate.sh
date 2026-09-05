#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_FILE="$ROOT_DIR/backend/db/queries/generation.sql"
GENERATED_FILE="$ROOT_DIR/backend/internal/platform/db/generation.sql.go"
TMP_DIR="$(mktemp -d)"

cleanup() {
  cp "$TMP_DIR/generation.sql" "$SOURCE_FILE"
  cp "$TMP_DIR/generation.sql.go" "$GENERATED_FILE"
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

cp "$SOURCE_FILE" "$TMP_DIR/generation.sql"
cp "$GENERATED_FILE" "$TMP_DIR/generation.sql.go"

cat >> "$SOURCE_FILE" <<'SQL'

-- name: SqlcDriftGateProbe :one
SELECT 1;
SQL

set +e
(
  cd "$ROOT_DIR"
  make sqlc-check >/tmp/synaudio-sqlc-drift-probe.log 2>&1
)
status=$?
set -e

if [[ $status -eq 0 ]]; then
  cat /tmp/synaudio-sqlc-drift-probe.log >&2
  echo "sqlc drift gate unexpectedly passed with intentionally stale generated output" >&2
  exit 1
fi

echo "sqlc drift gate correctly rejected intentionally stale generated output"
