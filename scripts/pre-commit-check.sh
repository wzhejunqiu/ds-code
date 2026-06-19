#!/usr/bin/env bash
# Pre-commit checks: gofmt + vet + golangci-lint for staged .go files only.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

staged=()
while IFS= read -r f; do
  [ -n "$f" ] && staged+=("$f")
done < <(git diff --cached --name-only --diff-filter=ACMR -- '*.go' || true)

if [ ${#staged[@]} -eq 0 ]; then
  exit 0
fi

unformatted=""
for f in "${staged[@]}"; do
  out=$(gofmt -l "$f" 2>/dev/null || true)
  if [ -n "$out" ]; then
    unformatted+="${out}"$'\n'
  fi
done
unformatted="${unformatted%"${unformatted##*[![:space:]]}"}"
if [ -n "$unformatted" ]; then
  echo "gofmt required on staged files:"
  echo "$unformatted"
  echo "run: gofmt -w ${unformatted//$'\n'/ }"
  exit 1
fi

pkgs=()
seen_dirs=""
for f in "${staged[@]}"; do
  d="$(dirname "$f")"
  case "$seen_dirs" in *"|${d}|"*) continue ;; esac
  seen_dirs="${seen_dirs}|${d}|"
  while IFS= read -r p; do
    [ -n "$p" ] && pkgs+=("$p")
  done < <(go list "./${d}/..." 2>/dev/null || go list "./${d}" 2>/dev/null || true)
done
# dedupe packages
if [ ${#pkgs[@]} -gt 0 ]; then
  pkgs=($(printf '%s\n' "${pkgs[@]}" | sort -u))
fi

if [ ${#pkgs[@]} -eq 0 ]; then
  exit 0
fi

exec "$ROOT/scripts/lint-packages.sh" "${pkgs[@]}"
