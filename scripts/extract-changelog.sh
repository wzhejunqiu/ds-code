#!/usr/bin/env bash
# Extract one Keep a Changelog section for GitHub Release body.
# Usage: extract-changelog.sh <version> [changelog-file]
#   version: 0.1.0 or v0.1.0
set -euo pipefail

VERSION="${1#v}"
FILE="${2:-CHANGELOG.md}"

if [ -z "$VERSION" ]; then
  echo "usage: $0 <version> [changelog-file]" >&2
  exit 1
fi
if [ ! -f "$FILE" ]; then
  echo "extract-changelog: $FILE not found" >&2
  exit 1
fi

awk -v ver="$VERSION" '
  function is_version_heading(line,    m) {
    return match(line, "^## \\[" ver "\\]( - [0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9])?$")
  }
  function is_footer_link(line) {
    return line ~ /^\[[0-9]+\.[0-9]+\.[0-9]+\]: /
  }
  BEGIN { found = 0; printed = 0 }
  {
    if (found && ($0 ~ /^## \[/ || is_footer_link($0))) {
      exit
    }
    if (!found && is_version_heading($0)) {
      found = 1
      print
      printed = 1
      next
    }
    if (found) {
      print
      printed = 1
    }
  }
  END {
    if (!printed) {
      printf "extract-changelog: no section ## [%s] in %s\n", ver, FILENAME > "/dev/stderr"
      exit 1
    }
  }
' "$FILE"
