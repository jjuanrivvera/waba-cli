<div align="center">

<img src="docs/assets/logo.svg" width="96" alt="waba logo">

# waba

[![CI](https://github.com/jjuanrivvera/waba-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/jjuanrivvera/waba-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/jjuanrivvera/waba-cli)](https://github.com/jjuanrivvera/waba-cli/releases/latest)
[![Coverage](https://img.shields.io/badge/coverage-%E2%89%A580%25-brightgreen)](https://github.com/jjuanrivvera/waba-cli/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/jjuanrivvera/waba-cli.svg)](https://pkg.go.dev/github.com/jjuanrivvera/waba-cli)
[![Go version](https://img.shields.io/github/go-mod/go-version/jjuanrivvera/waba-cli)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/jjuanrivvera/waba-cli)
[![Built with cliwright](https://img.shields.io/badge/built_with-cliwright-1f6feb)](https://cliwright.jjuanrivvera.com)

**WhatsApp Cloud API from the command line.**

[Documentation](https://jjuanrivvera.github.io/waba-cli/) · [Command reference](https://jjuanrivvera.github.io/waba-cli/commands/waba/)

</div>

`waba` is a complete command-line client for Meta's WhatsApp Cloud API: every message type,
media, phone numbers, message templates, the business profile, QR codes, WhatsApp Flows,
Calling, groups, analytics and account management — **102 of the 105 documented operations**,
enumerated in [`api-manifest.json`](api-manifest.json) and enforced by CI.

```console
$ waba send text --to 573001112233 "Your order shipped!"
MESSAGE_ID           WA_ID
wamid.HBgLNTcz...    573001112233
```

## Install

**macOS / Linux (one line):**

```sh
curl -fsSL https://raw.githubusercontent.com/jjuanrivvera/waba-cli/main/install.sh | sh
```

**Homebrew:**

```sh
brew install jjuanrivvera/waba-cli/waba-cli
```

**Windows (Scoop):**

```powershell
scoop bucket add waba-cli https://github.com/jjuanrivvera/scoop-waba-cli
scoop install waba-cli
```

**Go:**

```sh
go install github.com/jjuanrivvera/waba-cli/cmd/waba@latest
```

Debian/RPM/Alpine packages ship with every [release](https://github.com/jjuanrivvera/waba-cli/releases/latest).

## Quickstart

```sh
# One-time setup: token (hidden prompt), WABA id, default phone number.
# The wizard verifies the token and lists your phone numbers so you can pick one.
waba init

# Send things
waba send text     --to 573001112233 "Hola!"
waba send image    --to 573001112233 --link https://example.com/cat.jpg --caption "gato"
waba send template --to 573001112233 --name order_update --lang es_MX --param "Juan" --param "#1234"
waba send buttons  --to 573001112233 "Confirm your visit" --button yes:Sí --button no:No

# Manage templates
waba templates list --status APPROVED
waba templates create --data @welcome.json

# Inspect the account
waba phone list
waba analytics messaging --start 2026-08-01 --end 2026-08-17 --granularity DAY
waba doctor

# Anything the curated commands don't cover
waba api GET me -q fields=id,name
```

The token comes from a [System User](https://business.facebook.com/settings/system-users) in
Meta Business Manager with the `whatsapp_business_messaging` and
`whatsapp_business_management` permissions. It is stored in the OS keyring (macOS Keychain,
Linux Secret Service, Windows Credential Manager), with an AES-256-GCM encrypted-file
fallback for headless machines (`WABA_KEYRING_PASSWORD`).

## Highlights

- **The whole API.** 18 command groups over the Cloud API, Business Management API
  (templates, analytics, webhooks), Flows API, Calling API, Groups and the resumable Upload
  API. `waba <group> --help` shows runnable examples for everything.
- **Multi-account.** `--account` profiles bundle a WABA id, default phone number and app id
  — switch between businesses with one flag (`--profile` works too).
- **Pipeable output.** `-o table|json|yaml|csv|id`, `--columns`, a built-in `--jq` filter,
  and cursor-aware `--all` pagination. Notes go to stderr; stdout stays clean.
- **Safe by default.** Destructive commands confirm (or take `--yes`); `--dry-run` prints
  the exact `curl` equivalent with the token redacted; sends are never auto-retried, so a
  timeout can't double-message a customer.
- **Actionable errors.** Graph error codes map to fixes: a 131047 tells you to send a
  template; a 133010 tells you to register the number; every error carries its
  `fbtrace_id` for Meta support.
- **Rate-limit aware.** Fixed-rps pacing with halve-on-429 recovery, `Retry-After`
  honoured, and Meta's `X-Business-Use-Case-Usage` percentages fold into the limiter
  before you hit the hard block.
- **AI-agent ready.** `waba mcp` exposes every command as annotated MCP tools
  (read-only/write/destructive), and `waba agent guard` generates safety rails for
  Claude Code, Codex and OpenCode from the live command tree — irreversible operations are
  blocked, writes need approval. See [AI agents](https://jjuanrivvera.github.io/waba-cli/agents/).

## What it doesn't do

- **Webhooks are inbound.** Receiving messages requires a public HTTPS endpoint; `waba
  apps subscribe` manages the subscription, but serving the callback is your app's job.
- **Groups join-request moderation** (3 of the 105 enumerated operations) is not wrapped —
  Meta's parameter docs for it are not scrapable; `waba api` reaches it if you know the
  shape.
- **No first-party alternative exists to compare against**: Meta ships SDKs and a Postman
  collection for the Cloud API, but no official CLI.

## Environment variables

| Variable | Purpose |
|---|---|
| `WABA_ACCESS_TOKEN` | access token (overrides the keyring) |
| `WABA_ACCOUNT` | account (profile) to use |
| `WABA_WABA_ID` / `WABA_PHONE_NUMBER_ID` / `WABA_APP_ID` / `WABA_BUSINESS_ID` | id overrides |
| `WABA_GRAPH_VERSION` / `WABA_BASE_URL` | Graph version / host override |
| `WABA_KEYRING_PASSWORD` / `WABA_KEYRING_BACKEND` | encrypted-file credential store (headless) |

Precedence is always flag > environment > config file > default.

## Development

```sh
make build        # build to bin/waba
make check        # fmt + vet + lint + tests
make verify       # the full deterministic gate: check + spec gates + coverage + DoD
```

The command surface is generated from an enumerated API spec
([`specs/enumeration.json`](specs/enumeration.json)); `make verify` fails if the built CLI
diverges from it or covers less than 90% of the documented API. See
[AGENTS.md](AGENTS.md) for the architecture tour.

## License

[MIT](LICENSE)
