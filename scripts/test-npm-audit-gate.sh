#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATE="$ROOT_DIR/scripts/npm-audit-gate.sh"
MOCK_BIN_DIR="$(mktemp -d)"
TMP_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "$MOCK_BIN_DIR" "$TMP_DIR"
}
trap cleanup EXIT

expect_gate_failure() {
  local label="$1"
  shift

  set +e
  "$@" >"/tmp/synaudio-npm-audit-${label// /-}.log" 2>&1
  local status=$?
  set -e

  if [[ $status -eq 0 ]]; then
    cat "/tmp/synaudio-npm-audit-${label// /-}.log" >&2
    echo "npm audit gate unexpectedly passed: ${label}" >&2
    exit 1
  fi

  echo "npm audit gate correctly rejected: ${label}"
}

cat >"$MOCK_BIN_DIR/npm-empty-output" <<'EOF'
#!/usr/bin/env bash
exit 2
EOF

cat >"$MOCK_BIN_DIR/npm-malformed-json" <<'EOF'
#!/usr/bin/env bash
printf 'not-json'
exit 0
EOF

cat >"$MOCK_BIN_DIR/npm-error-json" <<'EOF'
#!/usr/bin/env bash
cat <<'JSON'
{
  "error": {
    "code": "EAI_AGAIN",
    "summary": "registry unreachable",
    "detail": "simulated npm audit registry failure for regression probe"
  }
}
JSON
exit 1
EOF

cat >"$MOCK_BIN_DIR/npm-missing-vulnerabilities" <<'EOF'
#!/usr/bin/env bash
cat <<'JSON'
{
  "auditReportVersion": 2,
  "metadata": {
    "vulnerabilities": {
      "info": 0,
      "low": 0,
      "moderate": 0,
      "high": 0,
      "critical": 0,
      "total": 0
    }
  }
}
JSON
exit 0
EOF

chmod +x "$MOCK_BIN_DIR"/*

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

expect_gate_failure "intentionally vulnerable lodash fixture" "$GATE" "$TMP_DIR"

for mock in \
  "empty output:NPM_AUDIT_GATE_NPM=$MOCK_BIN_DIR/npm-empty-output" \
  "malformed JSON:NPM_AUDIT_GATE_NPM=$MOCK_BIN_DIR/npm-malformed-json" \
  "error payload:NPM_AUDIT_GATE_NPM=$MOCK_BIN_DIR/npm-error-json" \
  "missing vulnerabilities object:NPM_AUDIT_GATE_NPM=$MOCK_BIN_DIR/npm-missing-vulnerabilities"
do
  label="${mock%%:*}"
  env_name="${mock#*:}"
  var="${env_name%%=*}"
  value="${env_name#*=}"
  expect_gate_failure "scanner failure (${label})" env "$var=$value" "$GATE" "$TMP_DIR"
done

echo "npm audit gate regression probes passed"
