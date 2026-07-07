#!/usr/bin/env bash
# Builds a universal (arm64+x86_64) libtokenizers.a for macOS desktop universal binaries.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$ROOT/third_party/tokenizers"
mkdir -p "$DEST"
WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

fetch_arch() {
  local arch="$1"
  local url="https://github.com/daulet/tokenizers/releases/latest/download/libtokenizers.darwin-${arch}.tar.gz"
  echo "Fetching $url"
  curl -fsSL -o "$WORKDIR/libtokenizers-${arch}.tar.gz" "$url"
  tar -xzf "$WORKDIR/libtokenizers-${arch}.tar.gz" -C "$WORKDIR"
  mv "$WORKDIR/libtokenizers.a" "$WORKDIR/libtokenizers-${arch}.a"
}

fetch_arch arm64
fetch_arch x86_64

lipo -create -output "$DEST/libtokenizers.a" "$WORKDIR/libtokenizers-arm64.a" "$WORKDIR/libtokenizers-x86_64.a"
if command -v ranlib >/dev/null 2>&1; then
  ranlib "$DEST/libtokenizers.a"
fi
echo "Installed universal $DEST/libtokenizers.a"
