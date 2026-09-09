# Dependency and CI supply-chain security

Synaudio treats dependency vulnerability scanning and third-party CI action integrity as explicit, fail-closed merge gates. This document defines the policy, tooling versions, triage semantics, and time-bounded exception process.

Release/deployment topology and artifact promotion remain owned by [#49](https://github.com/hoanghonghuy/synaudio/issues/49). That runbook should require these gates to be green before production promotion; this document does not duplicate release mechanics.

## CI gates

| Gate | Workflow job | Command | Scope |
|------|--------------|---------|-------|
| Go dependency vulnerabilities | `backend` | `scripts/govulncheck-gate.sh` | Production module graph (`backend/...`) |
| npm dependency vulnerabilities | `frontend` | `scripts/npm-audit-gate.sh` | Production dependencies only (`npm audit --omit=dev`) |
| sqlc drift | `backend` | `make sqlc-check` + `make sqlc-check-regression` | Unchanged (#46) |
| Test / Vet / Build / Typecheck | both jobs | existing steps | Unchanged |

Each gate has a regression script under `scripts/test-*-gate.sh` that proves an intentionally vulnerable fixture is rejected.

## Tooling versions (reproducible)

| Tool | Pinned version | Where enforced |
|------|----------------|----------------|
| Go toolchain | `1.26.6` (`backend/go.mod`, CI `setup-go`) | Compiler/stdlib vulnerability baseline |
| govulncheck | `v1.8.0` (`GOVULNCHECK_VERSION` in `scripts/govulncheck-gate.sh`, CI env) | Go advisory database scanner |
| Node.js | `22` (`.github/workflows/ci.yml`) | npm audit CLI |
| npm | Bundled with Node 22 | Lockfile audit against `frontend/package-lock.json` |

Identical source at the same commit should produce the same gate outcome until dependencies, advisories, or the pinned scanner/toolchain versions change intentionally.

## Go policy (`govulncheck`)

- Scan the full production backend module graph with symbol reachability (`govulncheck ./...`).
- Fail when **your code is affected** by a reported vulnerability (exit code non-zero).
- Indirect-only findings that do not affect reachable symbols are informational and do not fail the gate.
- Prefer upgrading dependencies and the Go toolchain over exceptions.

## npm policy (`npm audit`)

The gate is **fail-closed** for scanner and transport failures. It captures `npm audit` stdout, stderr, and exit code separately, validates the JSON shape (`auditReportVersion`, `vulnerabilities`, no top-level `error`), and fails when output is empty, malformed, or not an authoritative audit result. Only exit codes `0` (clean scan) and `1` (vulnerabilities reported in valid JSON) are accepted scanner outcomes; all other exit codes fail the gate.

Production dependency graph only (`--omit=dev`):

| Severity | Gate behavior |
|----------|---------------|
| `critical`, `high` | Fail |
| `moderate` | Fail when `fixAvailable` is true |
| `low`, `info` | Pass (review during routine dependency maintenance) |

Dev-only build tooling is out of scope for the merge gate because it does not ship in the production frontend artifact. Re-evaluate if the frontend build graph changes materially.

## CodeQL decision

CodeQL is **not** enabled in required CI for this repository at this time.

`govulncheck` covers Go dependency vulnerabilities with module-aware reachability, and the npm audit gate covers the committed frontend production graph. CodeQL would add static application security testing (SAST) overlap with limited additional dependency signal for the current Go + npm surface. Keeping required CI lean avoids parallel scanners with unclear ownership and branch-protection churn.

If SAST becomes a distinct requirement later, add a separate workflow with explicit scope, permissions, and required-check semantics rather than folding it into this dependency gate.

## GitHub Actions pinning

Third-party Actions referenced by required workflows are pinned to immutable commit SHAs with readable version comments (for example `actions/checkout@<sha> # v4.2.2`). Update pins deliberately with review; do not rely on floating `@v*` tags in required workflows.

## CI permissions

Required PR verification workflows use least privilege:

```yaml
permissions:
  contents: read
```

Do not add `write` permissions to ordinary PR verification solely to run scanners.

## Vulnerability exception process

Exceptions are recorded in [`vulnerability-exceptions.yaml`](./vulnerability-exceptions.yaml). **No silent allowlists** in scripts or undocumented `npm audit` ignore files. The gate scripts parse this file structurally via `scripts/lib/vulnerability-exceptions.mjs`: every entry must include all required fields, `expires_on` is enforced against the current UTC date, and npm exceptions suppress a finding only when `ecosystem`, `package`, installed `version`, and advisory `id` all match.

Every exception must include:

1. **Rationale** — why acceptance is temporarily justified.
2. **Owner** — accountable reviewer.
3. **Expiry** — `expires_on` review date; expired entries fail CI until renewed or removed.
4. **Affected dependency/version** — exact package/module and version.
5. **Mitigation** — compensating controls until upgrade.
6. **Upgrade path** — issue/PR/release plan.

Renewal requires a new review before extending `expires_on`. Remove exceptions immediately after upgrading the dependency.

## Triage playbook

1. Reproduce locally: `make govulncheck-check` and `make npm-audit-check` (or the underlying scripts).
2. Determine reachability: Go — read `govulncheck` symbol traces; npm — confirm production (`--omit=dev`) vs dev-only.
3. Prefer safe upgrades within the PR (dependency bump, Go patch release).
4. If upgrade is blocked, file a time-bounded exception in `vulnerability-exceptions.yaml` with owner and expiry.
5. For Actions pin updates, verify the tagged release commit SHA and run CI on the pin bump PR.

## Local commands

```bash
make govulncheck-check
make govulncheck-check-regression
make npm-audit-check
make npm-audit-check-regression
```

`scripts/pre-push.sh` runs both dependency gates before push when used as the local verification entrypoint.
