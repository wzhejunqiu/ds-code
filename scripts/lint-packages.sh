#!/usr/bin/env bash
# Run go vet and golangci-lint on the given Go import paths (e.g. from go list).
set -euo pipefail

if [ $# -eq 0 ]; then
  exit 0
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

GOLANGCI_LINT_VERSION="$(cat .golangci-lint-version)"
GOPATH_BIN="$(go env GOPATH)/bin"
GOLANGCI="$GOPATH_BIN/golangci-lint"

go vet -copylocks "$@"

if [ ! -x "$GOLANGCI" ]; then
  go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}"
fi

lint_paths=()
for pkg in "$@"; do
  rel="${pkg#github.com/wzhejunqiu/ds-code/}"
  if [ "$rel" = "$pkg" ]; then
    lint_paths+=("./${pkg}/...")
  else
    lint_paths+=("./${rel}/...")
  fi
done

"$GOLANGCI" run "${lint_paths[@]}"
