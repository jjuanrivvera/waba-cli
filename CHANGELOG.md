# Changelog

All notable changes to waba-cli are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to
[Semantic Versioning](https://semver.org/).

## [0.1.1] - 2026-08-17

### Fixed
- `waba init` now detects a phone number id pasted at the WABA id prompt, adopts it as the
  default phone number and points at where the real WABA id lives, instead of saving a
  broken account.
- Graph's "(#100) Tried accessing nonexisting field" errors now hint at the node-type
  mix-up (phone id vs WABA id vs app id) and the `metadata=1` probe.
- The encrypted-file keyring password check no longer refuses every file on Windows, where
  POSIX permission bits don't exist.
- `config view -o json`/`yaml` now renders accounts with stable lowercase keys.

### Security
- Toolchain bumped to go1.25.13, clearing GO-2026-6218 (net/url), GO-2026-6090
  (crypto/tls), GO-2026-6089 / GO-2026-5026 (net/http) and GO-2026-5972 (encoding/asn1).
- github.com/modelcontextprotocol/go-sdk bumped v1.3.0 → v1.4.1, clearing GO-2026-5771 and
  GO-2026-4773. `govulncheck ./...` is clean.

## [0.1.0] - 2026-08-17

### Added
- Full WhatsApp Cloud API surface: 102 of the 105 enumerated operations across 18 command
  groups — send (15 message types and interactive shapes), messages (read/typing), media,
  phone numbers, business profile, QR codes, blocklist, commerce, conversational
  components, templates, WABA account, webhook subscriptions, analytics (7 edges),
  resumable uploads, Flows, Calling, groups, and the Marketing Messages API.
- `--account` profiles (WABA id + phone number id + app id + Graph version) with the OS
  keyring for tokens and an AES-256-GCM encrypted-file fallback for headless machines.
- Meta commands: `auth`, `config`, `init` (wizard with phone-number discovery), `doctor`,
  `completion`, `alias`, `api` (raw escape hatch), `version`, `update` (self-update).
- Output formats table/json/yaml/csv/id, `--columns`, built-in `--jq`, cursor-aware
  `--all` pagination, `--dry-run` curl output with token redaction.
- Graph-aware resilience: idempotent-only retry with full jitter, `Retry-After` honoured,
  fixed-rps limiter with halve-on-429 and `X-Business-Use-Case-Usage` slowdown, error
  hints keyed to Graph error codes with `fbtrace_id` surfacing.
- AI-agent surface: `waba mcp` (annotated MCP tools via ophis) and `waba agent guard`
  (generated safety config for Claude Code, Codex and OpenCode, with an executable
  PreToolUse hook battery in the test suite).
- Enumerated API spec (`specs/enumeration.json`) generating `api-manifest.json`, with
  spec-consistency and spec-completeness CI gates.
