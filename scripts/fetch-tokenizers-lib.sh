#!/usr/bin/env bash
# Downloads the HuggingFace tokenizers static library for github.com/daulet/tokenizers.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$ROOT/third_party/tokenizers"
mkdir -p "$DEST"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  arm64|aarch64) arch="arm64" ;;
  x86_64|amd64) arch="x86_64" ;;
esac

url="https://github.com/daulet/tokenizers/releases/latest/download/libtokenizers.${os}-${arch}.tar.gz"
echo "Fetching $url"
curl -fsSL -o "$DEST/libtokenizers.tar.gz" "$url"
tar -xzf "$DEST/libtokenizers.tar.gz" -C "$DEST"
if command -v ranlib >/dev/null 2>&1; then
  ranlib "$DEST/libtokenizers.a"
fi
echo "Installed $DEST/libtokenizers.a"
