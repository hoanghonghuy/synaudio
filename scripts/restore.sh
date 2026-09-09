#!/usr/bin/env bash
# Restore a Synaudio PostgreSQL dump.
#
# Bounded local development (POSTGRES_PASSWORD + libpq vars; no DATABASE_URL):
#   POSTGRES_PASSWORD=... ./scripts/restore.sh <dump_file>
#
# URL-based/destructive restore requires all of (regardless of APP_ENV):
#   DATABASE_URL=<explicit recovery target; must equal ISOLATED_RECOVERY_DATABASE_URL>
#   ISOLATED_RECOVERY_DATABASE_URL=<designated isolated recovery target>
#   RECOVERY_TARGET=isolated
#   ALLOW_DESTRUCTIVE_RESTORE=YES_I_UNDERSTAND
#
# Optional fail-closed boundary when live production is configured:
#   PRODUCTION_DATABASE_URL=<live production target; must not equal ISOLATED_RECOVERY_DATABASE_URL>
#
# APP_ENV=production without DATABASE_URL is rejected. A supplied DATABASE_URL
# never authorizes destructive restore by itself; isolation + acknowledgement
# are always required for URL-based restore paths.

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

APP_ENV="${APP_ENV:-development}"
RESTORE_CONN_MODE=""

if [[ -n "${DATABASE_URL:-}" ]]; then
  if [[ "${RECOVERY_TARGET:-}" != "isolated" ]]; then
    echo "Error: DATABASE_URL restore is permitted only with RECOVERY_TARGET=isolated" >&2
    exit 1
  fi
  if [[ "${ALLOW_DESTRUCTIVE_RESTORE:-}" != "YES_I_UNDERSTAND" ]]; then
    echo "Error: destructive restore requires ALLOW_DESTRUCTIVE_RESTORE=YES_I_UNDERSTAND" >&2
    exit 1
  fi
  if [[ -z "${ISOLATED_RECOVERY_DATABASE_URL:-}" ]]; then
    echo "Error: DATABASE_URL restore requires ISOLATED_RECOVERY_DATABASE_URL" >&2
    exit 1
  fi
  if [[ -n "${PRODUCTION_DATABASE_URL:-}" ]] \
    && [[ "${ISOLATED_RECOVERY_DATABASE_URL}" == "${PRODUCTION_DATABASE_URL}" ]]; then
    echo "Error: ISOLATED_RECOVERY_DATABASE_URL must not equal PRODUCTION_DATABASE_URL" >&2
    exit 1
  fi
  if [[ "${DATABASE_URL}" != "${ISOLATED_RECOVERY_DATABASE_URL}" ]]; then
    echo "Error: DATABASE_URL must match ISOLATED_RECOVERY_DATABASE_URL for isolated recovery" >&2
    exit 1
  fi
  DB_URL="${DATABASE_URL}"
  RESTORE_CONN_MODE="connstring"
elif [[ "${APP_ENV}" == "production" ]]; then
  echo "Error: production restore requires explicit DATABASE_URL" >&2
  exit 1
else
  DEV_HOST="${POSTGRES_HOST:-localhost}"
  DEV_PORT="${POSTGRES_PORT:-5432}"
  DEV_DB="${POSTGRES_DB:-synaudio}"
  DEV_USER="${POSTGRES_USER:-synaudio}"
  if [[ -z "${POSTGRES_PASSWORD:-}" ]]; then
    echo "Error: local development restore requires POSTGRES_PASSWORD (DATABASE_URL triggers isolated restore gates)" >&2
    exit 1
  fi
  export PGPASSWORD="${POSTGRES_PASSWORD}"
  RESTORE_CONN_MODE="libpq"
fi

echo "Restoring from: ${DUMP_FILE}"
case "${DUMP_FILE}" in
  *.dump)
    if [[ "${RESTORE_CONN_MODE}" == "connstring" ]]; then
      pg_restore --dbname="${DB_URL}" --clean --if-exists --no-owner "${DUMP_FILE}"
    else
      pg_restore --host="${DEV_HOST}" --port="${DEV_PORT}" --username="${DEV_USER}" \
        --dbname="${DEV_DB}" --clean --if-exists --no-owner "${DUMP_FILE}"
    fi
    ;;
  *.sql)
    if [[ "${RESTORE_CONN_MODE}" == "connstring" ]]; then
      psql --dbname="${DB_URL}" --file="${DUMP_FILE}"
    else
      psql --host="${DEV_HOST}" --port="${DEV_PORT}" --username="${DEV_USER}" \
        --dbname="${DEV_DB}" --file="${DUMP_FILE}"
    fi
    ;;
  *)
    echo "Error: unrecognized dump extension (expected .dump or .sql): ${DUMP_FILE}" >&2
    exit 1
    ;;
esac

echo "Restore complete. Verify database + object-storage coherence before promotion."
