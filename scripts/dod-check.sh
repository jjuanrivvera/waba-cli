#!/usr/bin/env bash
# dod-check.sh — deterministic Definition-of-Done checks (cliwright GOAL.md §9/§12).
# One concrete check per atomic criterion. Copied into a generated CLI under scripts/.
# Usage: ./scripts/dod-check.sh <binary-name>
set -uo pipefail
BIN="${1:-__BINARY__}"
fail=0

ok()   { printf "  ✓ %s\n" "$1"; }
bad()  { printf "  ✗ %s\n" "$1"; fail=1; }
# have <description> <test-command...>
# The eval is deliberate: each check is written as a shell command *string* (with pipes,
# negation and globs), which is what makes the checks below readable one-liners.
# shellcheck disable=SC2294
have() { if eval "${@:2}" >/dev/null 2>&1; then ok "$1"; else bad "$1"; fi; }

echo "Definition-of-Done checks for '$BIN':"

# Agent surface
have "mcp server command present"        "rg -lq 'ophis|mcp' commands/mcp.go"
have "agent guard command present"       "test -f commands/agent.go"
have "guard PreToolUse hook generator"   "rg -lq 'PreToolUse' commands/agent_hosts.go"
have "guard hook execution battery"      "test -f commands/agent_hook_test.go"
have "guard hook path-prefix hardening"  "rg -Fq '([^[:space:]]*/)?' commands/agent_hosts.go"
have "guard no-jq JSON flattening"       "rg -Fq \"tr '\\n{}:,'\" commands/agent_hosts.go"

# Output formats (atomic — one per format)
for f in json yaml csv table; do
  have "output format: $f"               "rg -liq '\"$f\"|format$f|$f *format' internal/output"
done

# Resilience & safety
have "--dry-run prints equivalent curl"  "rg -lq 'dry-run' . && rg -lq 'curl' internal/api"
have "Ctrl-C: signal.NotifyContext"      "rg -lq 'signal.NotifyContext' cmd"
# A stray context.Background() in production code silently breaks Ctrl-C cancellation.
# Tests are excluded because a test has no cobra command to inherit a context from — the
# adjacent fmt.Scan* check already excludes them for the same reason.
have "no stray context.Background()"     "! rg -lq --glob '!*_test.go' 'context.Background()' commands internal/api"
have "secrets in OS keyring"             "rg -q 'zalando/go-keyring' go.mod"
# Interactive secret input must be hidden. fmt.Scan/Scanln/Scanf echo the secret to the
# terminal and stall on long pastes — read via promptSecret (term.ReadPassword) instead.
have "no plaintext stdin reads (fmt.Scan*)"  "! rg -lq --glob '!*_test.go' 'fmt\.Scan(ln|f)?\(' ."
have "idempotent-only retry"             "rg -lq 'idempotent|MethodGet|MethodPut|MethodDelete' internal/api"

# Meta commands (atomic — one per command)
for c in auth config init doctor completion alias api version; do
  have "meta command: $c"                "test -f commands/$c.go || rg -lq '\"$c\"' commands"
done

# Distribution & CI
have "GoReleaser config present"         "test -f .goreleaser.yaml || test -f .goreleaser.yml"
# If goreleaser is installed it MUST pass; if absent, skip (don't fake a pass with '|| true').
have "goreleaser check clean"            "! command -v goreleaser >/dev/null || goreleaser check"
have "install.sh present"                "test -f install.sh"
# If shellcheck is installed the installer MUST pass; if absent, skip.
have "install.sh shellcheck clean"       "! command -v shellcheck >/dev/null || ! test -f install.sh || shellcheck install.sh"
have "CI workflow present"               "test -f .github/workflows/ci.yml"
have "release workflow present"          "test -f .github/workflows/release.yml"

# Hygiene
for doc in README.md LICENSE CHANGELOG.md SECURITY.md AGENTS.md; do
  have "doc: $doc"                       "test -f $doc"
done
have "no committed token"                "! rg -lq '(api[_-]?key|token)\\s*[:=]\\s*[A-Za-z0-9_-]{16,}' --glob '!*.sh' --glob '!scripts/**' ."

if [[ $fail -ne 0 ]]; then
  echo "✗ Definition-of-Done incomplete"; exit 1
fi
echo "✓ Definition-of-Done satisfied"
