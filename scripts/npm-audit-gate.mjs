#!/usr/bin/env node

import process from 'node:process';
import {
  findMatchingNpmException,
  loadInstalledPackageVersions,
  loadValidatedExceptions,
} from './lib/vulnerability-exceptions.mjs';

function fail(message, detail) {
  console.error(`npm audit gate failed: ${message}`);
  if (detail) {
    console.error(detail);
  }
  process.exit(1);
}

function main() {
  const workdir = process.argv[2] ?? `${process.cwd()}/frontend`;
  const exceptionsFile =
    process.env.EXCEPTIONS_FILE ??
    `${process.cwd()}/docs/operations/vulnerability-exceptions.yaml`;
  const auditRaw = process.env.AUDIT_JSON ?? '';
  const auditStderr = process.env.AUDIT_STDERR ?? '';
  const auditExit = Number(process.env.AUDIT_EXIT);

  let registry;
  try {
    registry = loadValidatedExceptions(exceptionsFile);
  } catch (err) {
    fail('could not parse vulnerability exceptions registry', err.message);
  }

  if (registry.errors.length > 0) {
    fail('vulnerability exceptions registry is invalid or expired', registry.errors.join('\n'));
  }

  if (!auditRaw.trim()) {
    fail('no audit JSON output from npm audit', auditStderr.trim() || undefined);
  }

  let audit;
  try {
    audit = JSON.parse(auditRaw);
  } catch (err) {
    fail('malformed audit JSON from npm audit', `${err.message}\n${auditRaw.slice(0, 500)}`);
  }

  if (audit.error) {
    const error = audit.error;
    const code = error.code ?? 'UNKNOWN';
    const summary = error.summary ?? 'npm audit returned an error payload';
    const detail = error.detail ? `\n${error.detail}` : '';
    fail('npm audit did not return an authoritative result', `${code}: ${summary}${detail}`);
  }

  if (audit.auditReportVersion === undefined) {
    fail('audit JSON missing auditReportVersion');
  }

  if (typeof audit.vulnerabilities !== 'object' || audit.vulnerabilities === null || Array.isArray(audit.vulnerabilities)) {
    fail('audit JSON missing vulnerabilities object');
  }

  if (!Number.isInteger(auditExit) || (auditExit !== 0 && auditExit !== 1)) {
    fail(`unexpected npm audit exit code ${auditExit}`, auditStderr.trim() || undefined);
  }

  const vulnerabilityCount = Object.keys(audit.vulnerabilities).length;
  if (auditExit === 0 && vulnerabilityCount > 0) {
    fail('npm audit exit code 0 contradicts non-empty vulnerabilities payload');
  }
  if (auditExit === 1 && vulnerabilityCount === 0) {
    fail('npm audit exit code 1 without vulnerabilities or error payload');
  }

  const installedVersions = loadInstalledPackageVersions(workdir);
  const npmExceptions = registry.active.filter((exception) => exception.ecosystem === 'npm');
  const failures = [];

  for (const [name, detail] of Object.entries(audit.vulnerabilities)) {
    if (findMatchingNpmException(npmExceptions, detail, installedVersions)) {
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
}

main();
