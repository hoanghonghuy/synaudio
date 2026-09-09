#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: verify-storage-privacy.sh --unsigned-url URL

Negative post-deploy check: an unauthenticated direct object URL must not succeed.
Expect HTTP 403/404 (or connection failure for blocked endpoints).

  --unsigned-url URL   Direct object/bucket URL without auth/signature
EOF
}

UNSIGNED_URL=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --unsigned-url)
      UNSIGNED_URL="${2:-}"
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

if [[ -z "$UNSIGNED_URL" ]]; then
  echo "--unsigned-url is required" >&2
  usage >&2
  exit 2
fi

code="$(curl -sS -o /dev/null -w '%{http_code}' "$UNSIGNED_URL" || true)"

case "$code" in
  403|404|401)
    echo "OK   unsigned object access denied (HTTP $code)"
    ;;
  200|206)
    echo "FAIL unsigned object URL returned HTTP $code — bucket/object is publicly readable" >&2
    exit 1
    ;;
  *)
    echo "OK   unsigned object access not granted (HTTP $code)"
    ;;
esac
