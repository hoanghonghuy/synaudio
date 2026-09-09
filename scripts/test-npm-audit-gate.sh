#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

write_exceptions_file() {
  local path="$1"
  cat >"$path"
}

setup_lodash_probe() {
  local probe_dir="$TMP_DIR/lodash-probe"
  mkdir -p "$probe_dir"
  cat >"$probe_dir/package.json" <<'JSON'
{
  "name": "npm-audit-gate-probe",
  "private": true,
  "dependencies": {
    "lodash": "4.17.15"
  }
}
JSON
  (
    cd "$probe_dir"
    npm install --package-lock-only --ignore-scripts >/dev/null 2>&1
  )
  printf '%s' "$probe_dir"
}

run_gate() {
  local probe_dir="$1"
  local exceptions_file="$2"
  local log_file="$3"
  NPM_AUDIT_EXCEPTIONS_FILE="$exceptions_file" \
    bash "$ROOT_DIR/scripts/npm-audit-gate.sh" "$probe_dir" >"$log_file" 2>&1
}

expect_gate_failure() {
  local label="$1"
  local probe_dir="$2"
  local exceptions_file="$3"
  local log_file="$TMP_DIR/$(echo "$label" | tr ' /' '__').log"

  set +e
  run_gate "$probe_dir" "$exceptions_file" "$log_file"
  local status=$?
  set -e

  if [[ $status -eq 0 ]]; then
    echo "npm audit gate unexpectedly passed for probe: $label" >&2
    cat "$log_file" >&2
    exit 1
  fi

  echo "npm audit gate correctly rejected probe: $label"
}

PROBE_DIR="$(setup_lodash_probe)"
BASE_EXCEPTIONS="$TMP_DIR/exceptions-empty.yaml"
write_exceptions_file "$BASE_EXCEPTIONS" <<'YAML'
exceptions: []
YAML

expect_gate_failure "intentionally vulnerable lodash dependency" "$PROBE_DIR" "$BASE_EXCEPTIONS"

EXPIRED_EXCEPTIONS="$TMP_DIR/exceptions-expired.yaml"
write_exceptions_file "$EXPIRED_EXCEPTIONS" <<'YAML'
exceptions:
  - id: "1106913"
    ecosystem: npm
    package: lodash
    version: "4.17.15"
    severity: high
    rationale: Temporary acceptance for regression probe only.
    owner: platform-security
    expires_on: 2020-01-01
    mitigation: None; fixture only.
    upgrade_path: Remove after regression verification.
YAML

expect_gate_failure "expired exception must not suppress findings" "$PROBE_DIR" "$EXPIRED_EXCEPTIONS"

MALFORMED_EXCEPTIONS="$TMP_DIR/exceptions-malformed.yaml"
write_exceptions_file "$MALFORMED_EXCEPTIONS" <<'YAML'
exceptions:
  - id: "1106913"
    ecosystem: npm
    package: lodash
YAML

expect_gate_failure "malformed exception must not suppress findings" "$PROBE_DIR" "$MALFORMED_EXCEPTIONS"

MISMATCHED_EXCEPTIONS="$TMP_DIR/exceptions-mismatched.yaml"
write_exceptions_file "$MISMATCHED_EXCEPTIONS" <<'YAML'
exceptions:
  - id: "1106913"
    ecosystem: go
    package: left-pad
    version: "9.9.9"
    severity: high
    rationale: Deliberately mismatched scope for regression probe.
    owner: platform-security
    expires_on: 2099-12-31
    mitigation: None; fixture only.
    upgrade_path: Remove after regression verification.
YAML

expect_gate_failure "mismatched exception scope must not suppress findings" "$PROBE_DIR" "$MISMATCHED_EXCEPTIONS"

echo "npm audit gate regression probes passed"
