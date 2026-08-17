#!/usr/bin/env bash
# spec-check.sh — the determinism anchor (cliwright GOAL.md §11).
# The built CLI's command surface must match the spec-derived manifest, so two runs
# on the same API converge on the same surface. Copied into a generated CLI under scripts/.
# Usage: ./scripts/spec-check.sh [api-manifest.json]
set -uo pipefail
MANIFEST="${1:-api-manifest.json}"

[[ -f "$MANIFEST" ]] || { echo "✗ $MANIFEST missing — §11 requires a checked-in spec-derived manifest"; exit 1; }
BIN="$(jq -r '.binary // "__BINARY__"' "$MANIFEST")"
BIN_PATH="bin/$BIN"
[[ -x "$BIN_PATH" ]] || make build >/dev/null 2>&1 || { echo "✗ cannot build $BIN for the surface check"; exit 1; }

fail=0
# Every resource AND each of its declared verbs must be a reachable command — not just the
# resource (a resource missing `delete` must fail, or the manifest isn't really enforced).
while IFS=$'\t' read -r r verbs; do
  if ! "$BIN_PATH" "$r" --help >/dev/null 2>&1; then
    printf "  ✗ resource missing: %s\n" "$r"; fail=1; continue
  fi
  printf "  ✓ resource: %s\n" "$r"
  for v in $verbs; do
    if "$BIN_PATH" "$r" "$v" --help >/dev/null 2>&1; then
      printf "      ✓ %s %s\n" "$r" "$v"
    else
      printf "      ✗ %s missing verb: %s\n" "$r" "$v"; fail=1
    fi
  done
done < <(jq -r '.resources[] | "\(.name)\t\(.verbs // [] | join(" "))"' "$MANIFEST")

if [[ $fail -ne 0 ]]; then echo "✗ CLI surface diverges from $MANIFEST"; exit 1; fi
echo "✓ surface matches $MANIFEST"
