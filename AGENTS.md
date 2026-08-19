# AGENTS.md — working in the waba-cli repo

`waba` is a command-line client for the WhatsApp Cloud API: messaging, media, phone numbers,
templates, business profile, QR codes, flows, calling, groups, analytics and account
management, against Meta's Graph API. Built to the cliwright standard (Go + Cobra +
GoReleaser). This file orients an AI agent or human contributing to it.

## The one rule that matters

**`make verify` is the gate.** A change is done only when it exits `0`. It runs `make check`
(fmt, vet, golangci-lint, tests) + `spec-check` (the built surface matches
`api-manifest.json`) + `spec-completeness` (the manifest wraps ≥90% of the 105 enumerated
operations) + `cover-check` (≥80%) + `dod-check.sh`. Run the full `make verify` for anything
touching the command surface or a documented behaviour — not just `make check`.

## Architecture

- `specs/enumeration.json` — the enumerated API surface (one entry per
  method+path+distinctly-documented action, scraped from Meta's reference method index).
  `api-manifest.json` is GENERATED from it by `scripts/gen-manifest.sh`; never hand-edit the
  manifest. Surface changes = edit the enumeration, regenerate, implement, `make verify`.
- `internal/api/` — the Graph client: bearer auth, versioned URL building
  (`/{graph-version}/{path}`), idempotent-only retry honouring `Retry-After`, fixed-RPS +
  halve-on-429 rate limiting that also reads `X-Business-Use-Case-Usage` percentages,
  dry-run curl, `APIError` with Graph-code-keyed hints, cursor pagination, flexible JSON
  types, and one thin service file per resource group.
- `internal/auth/` — a single Bearer authenticator; OS keyring with an AES-256-GCM
  encrypted-file fallback (machine-keyed by default on headless boxes; a
  `WABA_KEYRING_PASSWORD` / `keyring-password` file upgrades it to password encryption).
- `internal/{config,output,version,update}` — `--account` profiles (WABA id + phone number
  id + app id + graph version) with manual precedence (no Viper), the table/json/yaml/csv/id
  renderer, build metadata, self-update.
- `commands/` — the cobra tree. `init()` appends builders to `registrars`/`metaRegistrars`;
  `NewRootCmd()` drains the queue onto a fresh root, which is what lets tests run in
  isolation. MCP annotations are stamped by `annotate()` as each command is built.
- `cmd/waba/main.go` — `signal.NotifyContext` (Ctrl-C cancels pagination and backoff) plus
  alias expansion before cobra parses.

## WhatsApp specifics you must not re-derive (see DECISIONS.md)

- **One path, many operations.** `POST /{phone-number-id}/messages` carries sends,
  mark-as-read, typing indicators, group pins and call-permission requests, discriminated by
  payload. The enumeration counts these separately; commands map 1:1 to enumeration verbs.
- **Never auto-retry POST** — a retried send double-messages a human. Idempotent methods only.
- **Resumable uploads** (`/upload:{session}`) use `Authorization: OAuth <token>` — not
  Bearer — plus a `file_offset` header. Handled by a per-request auth-scheme override.
- **Media downloads** go to a lookaside host with the same bearer token; the client
  restricts that host to facebook/fbcdn/fbsbx domains so a crafted response can't exfiltrate
  the token.
- **Granularity enums differ per analytics edge**: messaging uses `HALF_HOUR|DAY|MONTH`,
  conversation/pricing/call use `HALF_HOUR|DAILY|MONTHLY`.
- **Deleting a template by name removes every language variant**; by id needs BOTH
  `hsm_id` and `name`.
- **Groups join_requests are enumerated but not wrapped** (third-party-only param docs);
  reach them with `waba api`.

## House rules

- Comments explain **WHY**, not what.
- Thread `cmd.Context()` everywhere; never `context.Background()` (it breaks Ctrl-C).
- Secrets live in the keyring — never in config, code, or a commit message.
- Read secrets with `promptSecret` (raw mode), never `fmt.Scan*`.
- Pin ambiguous API assumptions in `DECISIONS.md`; read it back rather than re-deciding.
- Surface changes require updating `specs/enumeration.json`, regenerating the manifest
  (`scripts/gen-manifest.sh`) and the docs (`make docs-gen`), in the same commit.
- MCP exclusions match the **top-level group name only**; matching every node would drop
  `<resource> update` along with the self-updater.
- New commands ship with tests in the same commit; coverage is a ratchet.
