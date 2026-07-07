#!/usr/bin/env bash
# Verify Developer ID signature and notarization staple on a built .app bundle.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
APP="${1:-$ROOT/cmd/ds-code-desktop/bin/ds-code-desktop.app}"

if [[ ! -d "$APP" ]]; then
  echo "App bundle not found: $APP" >&2
  echo "Build first: cd cmd/ds-code-desktop && task darwin:package:universal" >&2
  exit 1
fi

echo "== codesign verify =="
codesign --verify --deep --strict --verbose=2 "$APP"

echo "== spctl assess =="
spctl -a -vv "$APP" || true

echo "== stapler validate =="
xcrun stapler validate "$APP" || echo "(not stapled yet — run task darwin:sign:notarize)"

echo "Done."
