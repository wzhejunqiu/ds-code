#!/usr/bin/env bash
# Check gofmt for all Go files (excluding third_party).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

unformatted=$(find . -name '*.go' -not -path './third_party/*' -print0 | xargs -0 gofmt -l 2>/dev/null || true)
if [ -n "$unformatted" ]; then
  echo "gofmt required on:"
  echo "$unformatted"
  echo "run: gofmt -w <files>"
  exit 1
fi
