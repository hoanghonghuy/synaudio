#!/usr/bin/env bash
# Backup the Synaudio PostgreSQL database to timestamped dump files.
#
# Development (requires POSTGRES_PASSWORD or DATABASE_URL):
#   POSTGRES_PASSWORD=... ./scripts/backup.sh [output_dir]
#   DATABASE_URL=... ./scripts/backup.sh [output_dir]
#
# Production safety contract:
#   APP_ENV=production DATABASE_URL=... BACKUP_OUTPUT_DIR=/secure/backup/path ./scripts/backup.sh
#
# Production never falls back to docker-compose credentials or the repository
# backup directory. The caller is responsible for moving the resulting bundle
# into encrypted, access-controlled backup storage according to the DR runbook.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ENV="${APP_ENV:-development}"

if [[ "${APP_ENV}" == "production" ]]; then
  if [[ -z "${DATABASE_URL:-}" ]]; then
    echo "Error: production backup requires explicit DATABASE_URL" >&2
    exit 1
  fi
  if [[ -z "${BACKUP_OUTPUT_DIR:-}" ]]; then
    echo "Error: production backup requires explicit BACKUP_OUTPUT_DIR" >&2
    exit 1
  fi
  if [[ $# -gt 0 ]]; then
    echo "Error: production backup target must be supplied via BACKUP_OUTPUT_DIR" >&2
    exit 1
  fi
  OUTPUT_DIR="${BACKUP_OUTPUT_DIR}"
else
  OUTPUT_DIR="${1:-${BACKUP_OUTPUT_DIR:-${SCRIPT_DIR}/../backups}}"
fi

mkdir -p "${OUTPUT_DIR}"
chmod 700 "${OUTPUT_DIR}" 2>/dev/null || true

TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
CUSTOM_FILE="${OUTPUT_DIR}/synaudio-${TIMESTAMP}.dump"
SQL_FILE="${OUTPUT_DIR}/synaudio-${TIMESTAMP}.sql"
MANIFEST_FILE="${OUTPUT_DIR}/synaudio-${TIMESTAMP}.manifest"

if [[ -n "${DATABASE_URL:-}" ]]; then
  echo "Using explicit DATABASE_URL for backup."
  pg_dump --dbname="${DATABASE_URL}" --format=custom --file="${CUSTOM_FILE}"
  pg_dump --dbname="${DATABASE_URL}" --file="${SQL_FILE}"
else
  HOST="${POSTGRES_HOST:-localhost}"
  PORT="${POSTGRES_PORT:-5432}"
  DB="${POSTGRES_DB:-synaudio}"
  USER="${POSTGRES_USER:-synaudio}"
  if [[ -z "${POSTGRES_PASSWORD:-}" ]]; then
    echo "Error: development backup requires POSTGRES_PASSWORD or DATABASE_URL" >&2
    exit 1
  fi
  export PGPASSWORD="${POSTGRES_PASSWORD}"

  echo "Development backup of ${DB}@${HOST}:${PORT} ..."
  pg_dump --host="${HOST}" --port="${PORT}" --username="${USER}" \
    --dbname="${DB}" --format=custom --file="${CUSTOM_FILE}"
  pg_dump --host="${HOST}" --port="${PORT}" --username="${USER}" \
    --dbname="${DB}" --file="${SQL_FILE}"
fi

chmod 600 "${CUSTOM_FILE}" "${SQL_FILE}" 2>/dev/null || true
CUSTOM_SHA256="$(sha256sum "${CUSTOM_FILE}" | awk '{print $1}')"
SQL_SHA256="$(sha256sum "${SQL_FILE}" | awk '{print $1}')"
cat >"${MANIFEST_FILE}" <<EOF
backup_timestamp_utc=${TIMESTAMP}
app_env=${APP_ENV}
custom_file=$(basename "${CUSTOM_FILE}")
custom_sha256=${CUSTOM_SHA256}
sql_file=$(basename "${SQL_FILE}")
sql_sha256=${SQL_SHA256}
EOF
chmod 600 "${MANIFEST_FILE}" 2>/dev/null || true

echo "Backup complete:"
echo "  custom:   ${CUSTOM_FILE}"
echo "  sql:      ${SQL_FILE}"
echo "  manifest: ${MANIFEST_FILE}"
