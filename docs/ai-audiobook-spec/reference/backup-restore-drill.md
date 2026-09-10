# Production Backup & Disaster-Recovery Contract

This is the canonical Synaudio durable-data recovery contract. It covers PostgreSQL **and** private object storage; restoring only one side is not a successful recovery.

## Service objectives

- Recovery point objective (RPO): **<= 24 hours**.
- Recovery time objective (RTO): **<= 8 hours** from declared recovery start until the isolated recovered stack passes all promotion checks.
- Record the source backup timestamp, incident/drill start, restore start/end, verification end, measured RPO/RTO, failures, and operator result for every drill.

## Durable backup scope

A recoverable point contains:

1. PostgreSQL: application/domain state, audit/provenance, listener state, job state, and audio metadata.
2. Private object storage: every production-persistent object namespace, including final audio/media and any other durable keys referenced by PostgreSQL.
3. A manifest identifying the database backup timestamp/checksums and the corresponding object-storage recovery point/version/snapshot identifier.

Temporary worker staging files are not durable backup authority.

## Security and retention

Production backup destinations must be private, encrypted at rest, and restricted to backup/recovery operators. Database dumps contain account/audit data and object backups contain private media; neither may be publicly readable. Transport to backup storage must be encrypted. Retention must provide at least one verified recovery point within the 24-hour RPO window plus an older recovery point suitable for rollback from a corrupt latest backup. Provider lifecycle/versioning/snapshot features may implement this policy, but the provider configuration must be recorded and tested.

Do not commit dumps, manifests, credentials, signed URLs, or recovery artifacts to Git.

## Database backup

The convenience path is deliberately local-only and uses libpq parameters with a loopback host (`localhost`, `127.0.0.1`, or `::1`):

```bash
POSTGRES_PASSWORD="$POSTGRES_PASSWORD" ./scripts/backup.sh ./backups
```

Any `DATABASE_URL` is treated as an arbitrary/non-local-capable target regardless of `APP_ENV` and therefore requires an explicit output destination. This prevents an omitted or mistyped environment label from dumping production data into the repository-local backup directory:

```bash
DATABASE_URL="$DATABASE_URL" \
BACKUP_OUTPUT_DIR=/secure/staging/synaudio \
./scripts/backup.sh
```

Production deliberately has no credential/target fallback and must use that explicit URL/output contract:

```bash
APP_ENV=production \
DATABASE_URL="$PRODUCTION_DATABASE_URL" \
BACKUP_OUTPUT_DIR=/secure/staging/synaudio \
./scripts/backup.sh
```

The script emits custom/plain dumps plus a SHA-256 manifest. Copy the resulting bundle into the encrypted backup destination, then remove local staging according to operator policy. A successful `pg_dump` alone is not the complete Synaudio backup.

## Object-storage backup

Use the storage provider's versioning/snapshot/replication/export mechanism, or a provider-neutral S3-compatible sync tool, to capture all durable production prefixes. Requirements:

- destination is private and encrypted;
- backup credentials are read-only on source where possible and write-only/least-privilege on destination;
- preserve object bytes and keys; preserve versions when the provider supports them;
- record provider-neutral recovery-point metadata (UTC time, source bucket, destination/snapshot identifier, object count/bytes where available) in the drill evidence;
- never use public bucket exposure as a backup mechanism.

The DB and object recovery points should be taken as close together as practical. If exact atomic snapshots across services are unavailable, recovery verification is authoritative: active/published DB references whose objects are absent make that candidate recovery point invalid for promotion.

## Restore: isolated target first

Never restore a drill directly over live production. Provision an isolated recovery database and private recovery bucket/object namespace first. Restore object bytes to the isolated target, then restore PostgreSQL against the compatible application/schema version.

Production-mode use of the repository restore script is intentionally fail-closed:

```bash
APP_ENV=production \
DATABASE_URL="$ISOLATED_RECOVERY_DATABASE_URL" \
ISOLATED_RECOVERY_DATABASE_URL="$ISOLATED_RECOVERY_DATABASE_URL" \
PRODUCTION_DATABASE_URL="$PRODUCTION_DATABASE_URL" \
RECOVERY_TARGET=isolated \
ALLOW_DESTRUCTIVE_RESTORE=YES_I_UNDERSTAND \
./scripts/restore.sh ./recovery/synaudio-<timestamp>.dump
```

`ISOLATED_RECOVERY_DATABASE_URL` is the designated isolated recovery target. The script refuses production-mode restore unless `DATABASE_URL` exactly matches that value. When `PRODUCTION_DATABASE_URL` is configured, the script also refuses an isolated target misconfigured to the same URL as live production.

The acknowledgement authorizes destruction of the **isolated target only**. Promotion/replacement of live production remains a separate explicit operator action after verification.

## Verification before recovery is declared successful

1. Verify dump SHA-256 against its manifest and verify the object backup/snapshot is readable.
2. Restore the database and object set into isolated targets.
3. Verify schema/migration compatibility with the application version being recovered; run the repository's normal migration/sqlc compatibility checks where applicable.
4. Run database integrity checks for critical relationships and domain invariants; do not rely on one sample row count.
5. Enumerate active/published durable audio/media references from PostgreSQL and verify every referenced object exists in the recovery bucket.
6. For newly generated audio carrying authoritative `sha256:<hex>` metadata, verify restored bytes against that checksum. Legacy rows without a checksum are explicitly **unverified**, not silently valid.
7. Detect/report orphaned durable objects separately. Orphans do not justify deleting data during the drill; cleanup requires its own reviewed policy.
8. Start the compatible API/worker/web against the isolated targets and require application health/readiness plus non-provider-critical smoke checks to pass.
9. Confirm private-storage access: anonymous direct reads fail while an authorized application retrieval path works for eligible content.
10. Record measured RPO/RTO and all verification evidence. Only a fully coherent candidate may be proposed for promotion.

If an active/published object is missing, checksum verification fails, the database backup is corrupt, schema compatibility fails, or required application readiness fails, mark that recovery point **FAILED**. Select an older coherent recovery point or repair through an explicitly reviewed recovery procedure; never promote a known partial restore as success.

## Promotion and destructive-production boundary

Promotion is an incident/operator decision, not part of the routine drill script. Before changing production, record approver, chosen recovery point, expected data loss/RPO, maintenance/write-freeze plan, rollback point, and exact database/object targets. Preserve the failed/live state until recovery is accepted when feasible. Application-image rollback is governed by #49; catastrophic durable-data recovery is governed here.

## Drill evidence template

Record at minimum:

- drill/incident ID and operator;
- backup DB timestamp + manifest SHA-256 values;
- object recovery-point/snapshot identifier and timestamp;
- recovery start, DB restore end, object restore end, verification end;
- measured RPO and RTO with PASS/FAIL against 24h/8h;
- schema/integrity/readiness results;
- referenced-object count checked, missing count, checksum verified/mismatch/unverified-legacy counts;
- private-access negative/authorized-path checks;
- failures/remediation and final PASS/FAIL decision.

Run this drill periodically and after material backup/topology changes. A backup policy that has never completed an isolated restore is not considered verified recovery capability.
