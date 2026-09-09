#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

cat >"$TMP_DIR/package.json" <<'JSON'
{
  "name": "npm-audit-gate-probe",
  "private": true,
  "dependencies": {
    "lodash": "4.17.15"
  }
}
JSON

(
  cd "$TMP_DIR"
  npm install --package-lock-only --ignore-scripts >/dev/null 2>&1
)

set +e
"$ROOT_DIR/scripts/npm-audit-gate.sh" "$TMP_DIR" >/tmp/synaudio-npm-audit-probe.log 2>&1
status=$?
set -e

if [[ $status -eq 0 ]]; then
  cat /tmp/synaudio-npm-audit-probe.log >&2
  echo "npm audit gate unexpectedly passed with intentionally vulnerable lodash version" >&2
  exit 1
fi

echo "npm audit gate correctly rejected intentionally vulnerable production dependency"
