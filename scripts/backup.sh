#!/usr/bin/env bash
# Backup the Synaudio PostgreSQL database to a timestamped dump file.
#
# Usage:
#   ./scripts/backup.sh [output_dir]
#
# Defaults:
#   output_dir = ./backups
#
# Environment (optional, falls back to docker-compose defaults):
#   DATABASE_URL  - full postgres connection string
#   POSTGRES_HOST - host (default: localhost)
#   POSTGRES_PORT - port (default: 5432)
#   POSTGRES_DB   - database name (default: synaudio)
#   POSTGRES_USER - user (default: synaudio)
#   POSTGRES_PASSWORD - password (default: synaudio)
#
# The dump is produced with pg_dump in custom format (-Fc) so it can be
# restored with pg_restore. A plain-text SQL fallback is also written for
# portability and manual inspection.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${1:-${SCRIPT_DIR}/../backups}"

mkdir -p "${OUTPUT_DIR}"

TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
CUSTOM_FILE="${OUTPUT_DIR}/synaudio-${TIMESTAMP}.dump"
SQL_FILE="${OUTPUT_DIR}/synaudio-${TIMESTAMP}.sql"

# Resolve connection parameters.
if [[ -n "${DATABASE_URL:-}" ]]; then
  # Use the full URL directly with pg_dump.
  echo "Using DATABASE_URL for backup."
  pg_dump --dbname="${DATABASE_URL}" --format=custom --file="${CUSTOM_FILE}"
  pg_dump --dbname="${DATABASE_URL}" --file="${SQL_FILE}"
else
  HOST="${POSTGRES_HOST:-localhost}"
  PORT="${POSTGRES_PORT:-5432}"
  DB="${POSTGRES_DB:-synaudio}"
  USER="${POSTGRES_USER:-synaudio}"
  export PGPASSWORD="${POSTGRES_PASSWORD:-synaudio}"

  echo "Backing up ${DB}@${HOST}:${PORT} ..."
  pg_dump --host="${HOST}" --port="${PORT}" --username="${USER}" \
    --dbname="${DB}" --format=custom --file="${CUSTOM_FILE}"
  pg_dump --host="${HOST}" --port="${PORT}" --username="${USER}" \
    --dbname="${DB}" --file="${SQL_FILE}"
fi

echo "Backup complete:"
echo "  custom: ${CUSTOM_FILE}"
echo "  sql:    ${SQL_FILE}"
