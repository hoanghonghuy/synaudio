#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
TMP_DIR="$(mktemp -d)"

cleanup() {
  cp "$TMP_DIR/go.mod" "$BACKEND_DIR/go.mod"
  cp "$TMP_DIR/go.sum" "$BACKEND_DIR/go.sum"
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

cp "$BACKEND_DIR/go.mod" "$TMP_DIR/go.mod"
cp "$BACKEND_DIR/go.sum" "$TMP_DIR/go.sum"

(
  cd "$BACKEND_DIR"
  go get github.com/jackc/pgx/v5@v5.7.5
)

set +e
"$ROOT_DIR/scripts/govulncheck-gate.sh" >/tmp/synaudio-govulncheck-probe.log 2>&1
status=$?
set -e

if [[ $status -eq 0 ]]; then
  cat /tmp/synaudio-govulncheck-probe.log >&2
  echo "govulncheck gate unexpectedly passed with intentionally vulnerable pgx version" >&2
  exit 1
fi

echo "govulncheck gate correctly rejected intentionally vulnerable dependency graph"
