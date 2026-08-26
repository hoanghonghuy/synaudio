#!/usr/bin/env bash
# Restore the Synaudio PostgreSQL database from a dump file.
#
# Usage:
#   ./scripts/restore.sh <dump_file>
#
# The dump file may be either:
#   - a custom-format dump (.dump) produced by pg_dump -Fc, restored with pg_restore
#   - a plain-text SQL dump (.sql), restored with psql
#
# Environment (optional, falls back to docker-compose defaults):
#   DATABASE_URL  - full postgres connection string
#   POSTGRES_HOST - host (default: localhost)
#   POSTGRES_PORT - port (default: 5432)
#   POSTGRES_DB   - database name (default: synaudio)
#   POSTGRES_USER - user (default: synaudio)
#   POSTGRES_PASSWORD - password (default: synaudio)
#
# WARNING: This DROPS and recreates the target database. Run only as part of a
# planned restore drill or disaster recovery, never against production without
# explicit approval.

set -euo pipefail

DUMP_FILE="${1:-}"
if [[ -z "${DUMP_FILE}" ]]; then
  echo "Usage: $0 <dump_file>" >&2
  exit 1
fi

if [[ ! -f "${DUMP_FILE}" ]]; then
  echo "Error: dump file not found: ${DUMP_FILE}" >&2
  exit 1
fi

# Resolve connection parameters.
if [[ -n "${DATABASE_URL:-}" ]]; then
  DB_URL="${DATABASE_URL}"
else
  HOST="${POSTGRES_HOST:-localhost}"
  PORT="${POSTGRES_PORT:-5432}"
  DB="${POSTGRES_DB:-synaudio}"
  USER="${POSTGRES_USER:-synaudio}"
  export PGPASSWORD="${POSTGRES_PASSWORD:-synaudio}"
  DB_URL="postgres://${USER}:${PGPASSWORD}@${HOST}:${PORT}/${DB}?sslmode=disable"
fi

echo "Restoring from: ${DUMP_FILE}"

case "${DUMP_FILE}" in
  *.dump)
    pg_restore --dbname="${DB_URL}" --clean --if-exists --no-owner "${DUMP_FILE}"
    ;;
  *.sql)
    psql --dbname="${DB_URL}" --file="${DUMP_FILE}"
    ;;
  *)
    echo "Error: unrecognized dump extension (expected .dump or .sql): ${DUMP_FILE}" >&2
    exit 1
    ;;
esac

echo "Restore complete."
