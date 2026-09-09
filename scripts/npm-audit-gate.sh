#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKDIR="${1:-$ROOT_DIR/frontend}"
EXCEPTIONS_FILE="${NPM_AUDIT_EXCEPTIONS_FILE:-$ROOT_DIR/docs/operations/vulnerability-exceptions.yaml}"
NPM_BIN="${NPM_AUDIT_GATE_NPM:-npm}"

cd "$WORKDIR"

audit_stderr_file="$(mktemp)"
audit_stdout_file="$(mktemp)"
cleanup() {
  rm -f "$audit_stderr_file" "$audit_stdout_file"
}
trap cleanup EXIT

set +e
"$NPM_BIN" audit --omit=dev --json >"$audit_stdout_file" 2>"$audit_stderr_file"
audit_exit=$?
set -e

export AUDIT_JSON="$(cat "$audit_stdout_file")"
export AUDIT_STDERR="$(cat "$audit_stderr_file")"
export AUDIT_EXIT="$audit_exit"
export EXCEPTIONS_FILE

node "$ROOT_DIR/scripts/npm-audit-gate.mjs" "$WORKDIR"
