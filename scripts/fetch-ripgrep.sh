#!/usr/bin/env bash
# Downloads ripgrep 15.1.0 release tarball for the current platform (no extract).
# Output: internal/tool/builtin/grep/rgbin/rg.tar.gz
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$ROOT/internal/tool/builtin/grep/rgbin"
RIPGREP_VERSION="${RIPGREP_VERSION:-15.1.0}"

mkdir -p "$DEST"

os="${RIPGREP_OS:-$(uname -s | tr '[:upper:]' '[:lower:]')}"
if [ -n "${RIPGREP_ARCH:-}" ]; then
  arch="$RIPGREP_ARCH"
else
  arch="$(uname -m)"
  case "$arch" in
  arm64 | aarch64) arch="aarch64" ;;
  x86_64 | amd64) arch="x86_64" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
  esac
fi

case "$os" in
darwin) target="${arch}-apple-darwin" ;;
linux) target="${arch}-unknown-linux-gnu" ;;
*)
  echo "unsupported OS: $os" >&2
  exit 1
  ;;
esac

archive="ripgrep-${RIPGREP_VERSION}-${target}.tar.gz"
url="https://github.com/BurntSushi/ripgrep/releases/download/${RIPGREP_VERSION}/${archive}"
out="$DEST/rg.tar.gz"

echo "Fetching $url"
curl -fsSL -o "$out" "$url"
echo "Installed $out"
