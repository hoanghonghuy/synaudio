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

AUDIT_JSON="$(cat "$audit_stdout_file")"
AUDIT_STDERR="$(cat "$audit_stderr_file")"

export AUDIT_JSON AUDIT_STDERR AUDIT_EXIT="$audit_exit" EXCEPTIONS_FILE

node <<'NODE'
const fs = require('node:fs');

const auditExit = Number(process.env.AUDIT_EXIT);
const auditRaw = process.env.AUDIT_JSON ?? '';
const auditStderr = process.env.AUDIT_STDERR ?? '';
const exceptionsFile = process.env.EXCEPTIONS_FILE;

function failScanner(message, detail) {
  console.error(`npm audit gate failed: ${message}`);
  if (detail) {
    console.error(detail);
  }
  process.exit(1);
}

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

if (!auditRaw.trim()) {
  failScanner('no audit JSON output from npm audit', auditStderr.trim() || undefined);
}

let audit;
try {
  audit = JSON.parse(auditRaw);
} catch (err) {
  failScanner('malformed audit JSON from npm audit', `${err.message}\n${auditRaw.slice(0, 500)}`);
}

if (audit.error) {
  const error = audit.error;
  const code = error.code ?? 'UNKNOWN';
  const summary = error.summary ?? 'npm audit returned an error payload';
  const detail = error.detail ? `\n${error.detail}` : '';
  failScanner('npm audit did not return an authoritative result', `${code}: ${summary}${detail}`);
}

if (audit.auditReportVersion === undefined) {
  failScanner('audit JSON missing auditReportVersion');
}

if (typeof audit.vulnerabilities !== 'object' || audit.vulnerabilities === null || Array.isArray(audit.vulnerabilities)) {
  failScanner('audit JSON missing vulnerabilities object');
}

if (!Number.isInteger(auditExit) || (auditExit !== 0 && auditExit !== 1)) {
  failScanner(`unexpected npm audit exit code ${auditExit}`, auditStderr.trim() || undefined);
}

const vulnerabilityCount = Object.keys(audit.vulnerabilities).length;
if (auditExit === 0 && vulnerabilityCount > 0) {
  failScanner('npm audit exit code 0 contradicts non-empty vulnerabilities payload');
}
if (auditExit === 1 && vulnerabilityCount === 0) {
  failScanner('npm audit exit code 1 without vulnerabilities or error payload');
}

const allowedExceptions = loadExceptionIds(exceptionsFile);
const vulns = audit.vulnerabilities;
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
