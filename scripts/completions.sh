#!/usr/bin/env bash
# completions.sh — generate shell completions into ./completions for packaging.
# Referenced by the goreleaser `before.hooks` and by the nfpms `contents`. Copied into a
# generated CLI under scripts/ (the goreleaser before hook calls `./scripts/completions.sh`).
set -euo pipefail
BIN="${1:-waba}"
mkdir -p completions
for sh in bash zsh fish; do
  go run ./cmd/"$BIN" completion "$sh" > "completions/$BIN.$sh"
done
echo "✓ wrote completions/$BIN.{bash,zsh,fish}"
