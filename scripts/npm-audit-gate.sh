#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKDIR="${1:-$ROOT_DIR/frontend}"
EXCEPTIONS_FILE="${NPM_AUDIT_EXCEPTIONS_FILE:-$ROOT_DIR/docs/operations/vulnerability-exceptions.yaml}"

cd "$WORKDIR"

audit_json="$(npm audit --omit=dev --json 2>/dev/null || true)"

AUDIT_JSON="$audit_json" EXCEPTIONS_FILE="$EXCEPTIONS_FILE" node <<'NODE'
const fs = require('node:fs');

const audit = JSON.parse(process.env.AUDIT_JSON || '{}');
const exceptionsFile = process.env.EXCEPTIONS_FILE;

function loadExceptionIds(path) {
  if (!fs.existsSync(path)) {
    return new Set();
  }
  const text = fs.readFileSync(path, 'utf8');
  const ids = new Set();
  for (const match of text.matchAll(/^\s*-\s*id:\s*["']?([^"'\n#]+)["']?\s*$/gm)) {
    ids.add(match[1].trim());
  }
  return ids;
}

const allowedExceptions = loadExceptionIds(exceptionsFile);
const vulns = audit.vulnerabilities ?? {};
const failures = [];

for (const [name, detail] of Object.entries(vulns)) {
  const via = detail.via ?? [];
  const advisoryIds = via
    .map((entry) => (typeof entry === 'string' ? entry : entry?.source))
    .filter(Boolean);

  if (allowedExceptions.has(name) || advisoryIds.some((id) => allowedExceptions.has(String(id)))) {
    continue;
  }

  const severity = detail.severity;
  const fixAvailable = Boolean(
    detail.fixAvailable === true ||
      (typeof detail.fixAvailable === 'object' && detail.fixAvailable !== null)
  );

  if (severity === 'critical' || severity === 'high') {
    failures.push({ name, severity, reason: 'production dependency at or above high severity' });
    continue;
  }

  if (severity === 'moderate' && fixAvailable) {
    failures.push({ name, severity, reason: 'moderate production vulnerability with an available fix' });
  }
}

if (failures.length > 0) {
  console.error('npm audit gate failed for production dependencies (--omit=dev):');
  for (const failure of failures) {
    console.error(`- ${failure.name}: ${failure.severity} (${failure.reason})`);
  }
  process.exit(1);
}

console.log('npm audit gate passed for production dependencies (--omit=dev)');
NODE
