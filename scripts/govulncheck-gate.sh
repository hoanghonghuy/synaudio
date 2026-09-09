#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOVULNCHECK_VERSION="${GOVULNCHECK_VERSION:-v1.8.0}"
BACKEND_DIR="$ROOT_DIR/backend"

cd "$BACKEND_DIR"
export GOTOOLCHAIN=local
go install "golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}"
"$(go env GOPATH)/bin/govulncheck" ./...
