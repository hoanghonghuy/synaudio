#!/usr/bin/env bash
# Hermetic regression for production restore isolation enforcement.
# Mocks pg_restore/psql on PATH so no real Postgres is required.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RESTORE_SCRIPT="$ROOT_DIR/scripts/restore.sh"
TMP_DIR="$(mktemp -d)"
MOCK_BIN="$TMP_DIR/mock-bin"
DUMP_FILE="$TMP_DIR/test.dump"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

mkdir -p "$MOCK_BIN"
touch "$DUMP_FILE"

write_mock() {
  local name="$1"
  local body="$2"
  cat >"$MOCK_BIN/$name" <<EOF
#!/usr/bin/env bash
$body
EOF
  chmod +x "$MOCK_BIN/$name"
}

write_mock pg_restore 'echo "UNEXPECTED: pg_restore was invoked" >&2; exit 99'
write_mock psql 'echo "UNEXPECTED: psql was invoked" >&2; exit 99'

export PATH="$MOCK_BIN:$PATH"

ISOLATED_URL="postgres://recovery@recovery-host:5432/synaudio_recovery?sslmode=require"
PRODUCTION_URL="postgres://live@prod-host:5432/synaudio?sslmode=require"

expect_fail() {
  local expected_substring="$1"
  shift
  local output
  local status

  set +e
  output="$(env "$@" bash "$RESTORE_SCRIPT" "$DUMP_FILE" 2>&1)"
  status=$?
  set -e

  if [[ $status -eq 0 ]]; then
    echo "Expected failure but restore succeeded: $output" >&2
    exit 1
  fi
  if [[ "$output" != *"$expected_substring"* ]]; then
    echo "Expected output containing '$expected_substring', got:" >&2
    echo "$output" >&2
    exit 1
  fi
}

expect_success() {
  local output
  local status

  set +e
  output="$(env "$@" bash "$RESTORE_SCRIPT" "$DUMP_FILE" 2>&1)"
  status=$?
  set -e

  if [[ $status -ne 0 ]]; then
    echo "Expected success but restore failed: $output" >&2
    exit 1
  fi
}

# Omitted or mistyped APP_ENV must not bypass URL-based destructive restore gates.
expect_fail "RECOVERY_TARGET=isolated" \
  DATABASE_URL="$PRODUCTION_URL"

expect_fail "RECOVERY_TARGET=isolated" \
  APP_ENV=development \
  DATABASE_URL="$PRODUCTION_URL"

expect_fail "RECOVERY_TARGET=isolated" \
  APP_ENV=develoment \
  DATABASE_URL="$PRODUCTION_URL"

# Caller flags alone cannot authorize a live production target.
expect_fail "ISOLATED_RECOVERY_DATABASE_URL" \
  APP_ENV=production \
  DATABASE_URL="$PRODUCTION_URL" \
  RECOVERY_TARGET=isolated \
  ALLOW_DESTRUCTIVE_RESTORE=YES_I_UNDERSTAND

expect_fail "must match ISOLATED_RECOVERY_DATABASE_URL" \
  APP_ENV=production \
  DATABASE_URL="$PRODUCTION_URL" \
  ISOLATED_RECOVERY_DATABASE_URL="$ISOLATED_URL" \
  RECOVERY_TARGET=isolated \
  ALLOW_DESTRUCTIVE_RESTORE=YES_I_UNDERSTAND

expect_fail "must not equal PRODUCTION_DATABASE_URL" \
  APP_ENV=production \
  DATABASE_URL="$PRODUCTION_URL" \
  ISOLATED_RECOVERY_DATABASE_URL="$PRODUCTION_URL" \
  PRODUCTION_DATABASE_URL="$PRODUCTION_URL" \
  RECOVERY_TARGET=isolated \
  ALLOW_DESTRUCTIVE_RESTORE=YES_I_UNDERSTAND

# Omitted or mistyped APP_ENV must not bypass libpq restore to remote hosts.
expect_fail "localhost" \
  POSTGRES_HOST=prod-host \
  POSTGRES_PASSWORD=secret

expect_fail "localhost" \
  APP_ENV=development \
  POSTGRES_HOST=prod-host \
  POSTGRES_PASSWORD=secret

expect_fail "localhost" \
  APP_ENV=develoment \
  POSTGRES_HOST=prod-host \
  POSTGRES_PASSWORD=secret

write_mock pg_restore 'exit 0'
write_mock psql 'exit 0'

expect_success \
  APP_ENV=production \
  DATABASE_URL="$ISOLATED_URL" \
  ISOLATED_RECOVERY_DATABASE_URL="$ISOLATED_URL" \
  PRODUCTION_DATABASE_URL="$PRODUCTION_URL" \
  RECOVERY_TARGET=isolated \
  ALLOW_DESTRUCTIVE_RESTORE=YES_I_UNDERSTAND

# Isolated URL restore succeeds without APP_ENV=production when fully acknowledged.
expect_success \
  DATABASE_URL="$ISOLATED_URL" \
  ISOLATED_RECOVERY_DATABASE_URL="$ISOLATED_URL" \
  PRODUCTION_DATABASE_URL="$PRODUCTION_URL" \
  RECOVERY_TARGET=isolated \
  ALLOW_DESTRUCTIVE_RESTORE=YES_I_UNDERSTAND

# Bounded local dev path: POSTGRES_PASSWORD + allowlisted libpq host, no DATABASE_URL.
expect_success \
  POSTGRES_PASSWORD=localdev

expect_success \
  POSTGRES_HOST=127.0.0.1 \
  POSTGRES_PASSWORD=localdev

expect_success \
  POSTGRES_HOST=::1 \
  POSTGRES_PASSWORD=localdev

echo "restore isolation gate regression passed"
