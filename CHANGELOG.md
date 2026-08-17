# Changelog

All notable changes to waba-cli are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to
[Semantic Versioning](https://semver.org/).

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
