#!/usr/bin/env bash
# Hermetic regression for backup target/output safety enforcement.
# Mocks pg_dump on PATH so no real Postgres is required.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_SCRIPT="$ROOT_DIR/scripts/backup.sh"
TMP_DIR="$(mktemp -d)"
MOCK_BIN="$TMP_DIR/mock-bin"
CALL_LOG="$TMP_DIR/pg_dump.calls"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

mkdir -p "$MOCK_BIN"

cat >"$MOCK_BIN/pg_dump" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
: "${CALL_LOG:?}"
printf '%s\n' "$*" >>"$CALL_LOG"
out=""
for arg in "$@"; do
  case "$arg" in
    --file=*) out="${arg#--file=}" ;;
  esac
done
if [[ -z "$out" ]]; then
  echo "mock pg_dump missing --file" >&2
  exit 98
fi
mkdir -p "$(dirname "$out")"
printf 'mock-backup\n' >"$out"
EOF
chmod +x "$MOCK_BIN/pg_dump"

export PATH="$MOCK_BIN:$PATH"
export CALL_LOG

REMOTE_URL="postgres://backup@prod-host:5432/synaudio?sslmode=require"

expect_fail_without_dump() {
  local expected_substring="$1"
  shift
  local output status before after
  before="$(wc -l <"$CALL_LOG" 2>/dev/null || printf '0')"
  set +e
  output="$(env "$@" bash "$BACKUP_SCRIPT" 2>&1)"
  status=$?
  set -e
  after="$(wc -l <"$CALL_LOG" 2>/dev/null || printf '0')"
  if [[ $status -eq 0 ]]; then
    echo "Expected backup failure but succeeded: $output" >&2
    exit 1
  fi
  if [[ "$output" != *"$expected_substring"* ]]; then
    echo "Expected output containing '$expected_substring', got:" >&2
    echo "$output" >&2
    exit 1
  fi
  if [[ "$before" != "$after" ]]; then
    echo "pg_dump was invoked despite fail-closed gate" >&2
    exit 1
  fi
}

expect_success() {
  local output status
  set +e
  output="$(env "$@" bash "$BACKUP_SCRIPT" 2>&1)"
  status=$?
  set -e
  if [[ $status -ne 0 ]]; then
    echo "Expected backup success but failed: $output" >&2
    exit 1
  fi
}

: >"$CALL_LOG"

# Omitted/mistyped APP_ENV must not allow an arbitrary URL to fall back to a
# repository-local output directory.
expect_fail_without_dump "BACKUP_OUTPUT_DIR" \
  DATABASE_URL="$REMOTE_URL"

expect_fail_without_dump "BACKUP_OUTPUT_DIR" \
  APP_ENV=develoment \
  DATABASE_URL="$REMOTE_URL"

# The libpq convenience path is positively local-only.
expect_fail_without_dump "localhost" \
  POSTGRES_HOST=prod-host \
  POSTGRES_PASSWORD=secret

expect_fail_without_dump "localhost" \
  APP_ENV=development \
  POSTGRES_HOST=prod-host \
  POSTGRES_PASSWORD=secret

expect_fail_without_dump "localhost" \
  APP_ENV=develoment \
  POSTGRES_HOST=prod-host \
  POSTGRES_PASSWORD=secret

# Exact production mode cannot use the local convenience path either.
expect_fail_without_dump "production backup requires explicit DATABASE_URL" \
  APP_ENV=production \
  POSTGRES_PASSWORD=secret

# Explicit URL + explicit output is the supported arbitrary/non-local path.
expect_success \
  DATABASE_URL="$REMOTE_URL" \
  BACKUP_OUTPUT_DIR="$TMP_DIR/url-output"

# Bounded local development remains convenient on loopback hosts.
expect_success \
  POSTGRES_PASSWORD=localdev \
  BACKUP_OUTPUT_DIR="$TMP_DIR/local-default"

expect_success \
  POSTGRES_HOST=127.0.0.1 \
  POSTGRES_PASSWORD=localdev \
  BACKUP_OUTPUT_DIR="$TMP_DIR/local-v4"

expect_success \
  POSTGRES_HOST=::1 \
  POSTGRES_PASSWORD=localdev \
  BACKUP_OUTPUT_DIR="$TMP_DIR/local-v6"

# Every successful backup invokes pg_dump twice and produces a manifest.
if [[ "$(wc -l <"$CALL_LOG")" -ne 8 ]]; then
  echo "Expected 8 pg_dump calls from four successful backups" >&2
  exit 1
fi
for dir in "$TMP_DIR/url-output" "$TMP_DIR/local-default" "$TMP_DIR/local-v4" "$TMP_DIR/local-v6"; do
  if ! compgen -G "$dir/*.manifest" >/dev/null; then
    echo "Expected backup manifest in $dir" >&2
    exit 1
  fi
done

echo "backup target gate regression passed"
