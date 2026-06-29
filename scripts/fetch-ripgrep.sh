#!/usr/bin/env bash
# Downloads ripgrep 15.1.0 release tarball for the current platform (no extract).
# Output: internal/tool/builtin/grep/rgbin/rg.tar.gz
#
# Asset mapping follows https://github.com/BurntSushi/ripgrep/releases/tag/15.1.0
# (Linux x86_64 is musl-only; Linux aarch64 is gnu-only).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$ROOT/internal/tool/builtin/grep/rgbin"
RIPGREP_VERSION="${RIPGREP_VERSION:-15.1.0}"

mkdir -p "$DEST"

normalize_arch() {
  case "$1" in
  arm64 | aarch64) echo aarch64 ;;
  x86_64 | amd64) echo x86_64 ;;
  *) echo "$1" ;;
  esac
}

os="${RIPGREP_OS:-$(uname -s | tr '[:upper:]' '[:lower:]')}"
arch="$(normalize_arch "${RIPGREP_ARCH:-$(uname -m)}")"

case "$os" in
darwin)
  case "$arch" in
  aarch64 | x86_64) target="${arch}-apple-darwin" ;;
  *)
    echo "unsupported darwin arch: $arch" >&2
    exit 1
    ;;
  esac
  ;;
linux)
  case "$arch" in
  x86_64) target="x86_64-unknown-linux-musl" ;;
  aarch64) target="aarch64-unknown-linux-gnu" ;;
  *)
    echo "unsupported linux arch: $arch" >&2
    exit 1
    ;;
  esac
  ;;
*)
  echo "unsupported OS: $os" >&2
  exit 1
  ;;
esac

archive="ripgrep-${RIPGREP_VERSION}-${target}.tar.gz"
url="https://github.com/BurntSushi/ripgrep/releases/download/${RIPGREP_VERSION}/${archive}"
out="$DEST/rg.tar.gz"
stamp="$DEST/.ripgrep-target"

if [ -f "$out" ] && [ -f "$stamp" ] && [ "$(cat "$stamp")" = "$target" ]; then
  echo "Already present: $out ($target)"
  exit 0
fi

echo "Fetching $url"
curl -fsSL -o "$out" "$url"
echo "$target" >"$stamp"
echo "Installed $out ($target)"
