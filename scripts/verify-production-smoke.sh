#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: verify-production-smoke.sh --api-base URL [--worker-probe URL]

Post-deploy smoke checks for Synaudio production rollout.
Does not invoke live AI/TTS generation.

  --api-base URL       Public API/web base (e.g. https://app.example.com)
  --worker-probe URL   Worker probe base reachable by this host (default: skip worker checks)
EOF
}

API_BASE=""
WORKER_PROBE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --api-base)
      API_BASE="${2:-}"
      shift 2
      ;;
    --worker-probe)
      WORKER_PROBE="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$API_BASE" ]]; then
  echo "--api-base is required" >&2
  usage >&2
  exit 2
fi

API_BASE="${API_BASE%/}"

check_json_status() {
  local url="$1"
  local expect_field="$2"
  local expect_value="$3"
  local body
  body="$(curl -fsS "$url")"
  local status
  status="$(printf '%s' "$body" | python3 -c "import json,sys; print(json.load(sys.stdin).get('$expect_field',''))")"
  if [[ "$status" != "$expect_value" ]]; then
    echo "FAIL $url expected $expect_field=$expect_value got $status body=$body" >&2
    exit 1
  fi
  echo "OK   $url ($expect_field=$expect_value)"
}

echo "== API liveness =="
check_json_status "$API_BASE/health" status ok

echo "== API readiness =="
ready_body="$(curl -fsS "$API_BASE/ready")"
ready_status="$(printf '%s' "$ready_body" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("status",""))')"
if [[ "$ready_status" != "ready" ]]; then
  echo "FAIL $API_BASE/ready expected status=ready got $ready_status body=$ready_body" >&2
  exit 1
fi
echo "OK   $API_BASE/ready (status=ready)"

echo "== Web shell =="
curl -fsS -o /dev/null "$API_BASE/"
echo "OK   $API_BASE/ (HTTP 200)"

if [[ -n "$WORKER_PROBE" ]]; then
  WORKER_PROBE="${WORKER_PROBE%/}"
  echo "== Worker liveness =="
  check_json_status "$WORKER_PROBE/health" status ok
  echo "== Worker readiness =="
  check_json_status "$WORKER_PROBE/ready" status ready
fi

echo "All production smoke checks passed."
