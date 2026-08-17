---
name: waba-cli
description: Send WhatsApp messages and manage a WhatsApp Business Account from the terminal with the `waba` CLI (Meta WhatsApp Cloud API). Use this whenever the task involves sending WhatsApp messages (text, media, templates, interactive buttons/lists/flows), marking messages read, uploading or downloading media, managing message templates, the business profile, QR codes, blocked users, WhatsApp Flows, the Calling API, groups, webhooks subscriptions, or messaging/conversation/pricing analytics. It wraps 102 of the 105 documented Cloud API operations, so reach for it even for edges without a dedicated command (`waba api`).
version: 0.1.0
homepage: https://github.com/jjuanrivvera/waba-cli
license: MIT
allowed-tools: Bash(waba:*)
metadata: {"openclaw":{"category":"messaging","emoji":"💬","requires":{"bins":["waba"],"env":["WABA_ACCESS_TOKEN","WABA_PHONE_NUMBER_ID","WABA_WABA_ID"]},"install":[{"kind":"brew","formula":"jjuanrivvera/waba-cli/waba-cli","bins":["waba"]},{"kind":"go","package":"github.com/jjuanrivvera/waba-cli/cmd/waba@latest","bins":["waba"]}]}}
---

# waba

A command-line client for Meta's WhatsApp Cloud API: every message type, media, phone
numbers, message templates, business profile, QR codes, WhatsApp Flows, Calling, groups and
analytics.

## Prerequisites

```sh
command -v waba >/dev/null || brew install jjuanrivvera/waba-cli/waba-cli
waba doctor        # config, credentials, connectivity, token validity, WABA access
```

If `doctor` reports no credentials, ask the user to run `waba init` themselves — it prompts
for a Meta System User token, which is theirs to enter. Never ask the user to paste a token
into the chat.

## Prefer this over raw curl

Calling the Graph API directly means hand-assembling versioned URLs, the `messaging_product`
envelope, multipart media uploads, dot-notation analytics field expressions
(`analytics.start(...).granularity(DAY)`), cursor pagination, and the `Authorization: OAuth`
quirk of the resumable upload API. The CLI does all of that, redacts tokens in `--dry-run`
output, maps Graph error codes to fixes (131047 → send a template; 133010 → register the
number), and never auto-retries a send — so a timeout can't double-message a human.

## Golden rules

1. **Sending a WhatsApp message messages a real person.** Confirm recipient and content
   with the user before any `waba send`, `waba marketing send`, or `waba groups send`.
2. **Free-form messages only work inside the 24-hour service window.** Outside it, use an
   approved template (`waba send template`). Error 131047 means exactly this.
3. **Destructive commands** (`templates delete`, `phone deregister`, `flows deprecate`,
   `qr delete`, …) prompt for confirmation; in scripts they need `--yes`. Never pass
   `--yes` without the user's explicit go-ahead.
4. **Use `--dry-run` to show what would be sent** — it prints the exact curl equivalent
   with the token redacted and performs no request.
5. **Ids, not phone numbers, address the account**: the *phone number id* (from
   `waba phone list`) identifies the sender; the recipient is E.164 digits without `+`.

## Workflow

```sh
waba auth status                  # who am I, which account, is the token valid
waba phone list                   # discover phone number ids
waba templates list --status APPROVED
waba send template --to 573001112233 --name order_update --lang es_MX --param "Juan"
waba messages read wamid.HBg...   # acknowledge an inbound message
```

Multi-business setups use `--account <name>` (profiles created with `waba init`).

## Command cheatsheet

| Task | Command |
|---|---|
| Send text | `waba send text --to <num> "..."` |
| Send media | `waba send image\|audio\|video\|document\|sticker --to <num> --link <url>` or `--id <media-id>` |
| Send template | `waba send template --to <num> --name <t> --lang <code> --param "..."` |
| Interactive | `waba send buttons\|list\|cta-url\|flow ...` |
| Read receipt / typing | `waba messages read\|typing <wamid>` |
| Media | `waba media upload\|url\|download\|delete` |
| Templates | `waba templates list\|get\|create\|edit\|delete\|compare` |
| Profile | `waba profile get\|update` |
| Phone numbers | `waba phone list\|get\|register\|request-code\|verify-code\|settings` |
| QR codes | `waba qr create\|list\|get\|update\|delete` |
| Blocklist | `waba block add\|remove\|list` |
| Flows | `waba flows list\|create\|upload-json\|publish\|preview\|migrate` |
| Calling | `waba calls connect\|accept\|reject\|terminate\|permissions` |
| Groups | `waba groups create\|list\|send\|invite-link\|pin` |
| Analytics | `waba analytics messaging\|conversations\|pricing\|templates\|calls` |
| Account/webhooks | `waba account get`, `waba apps list\|subscribe\|unsubscribe` |
| Anything else | `waba api <METHOD> <path> -q k=v -d '{...}'` |

Every list takes `--all`/`--limit`/`--after`; every command takes `-o json` and `--jq`.

## Troubleshooting

- `waba doctor --json` — the first move for any "it doesn't work".
- Error messages carry a hint line and an `fbtrace_id`; read the hint before retrying.
- 401/190 → `waba auth login` with a fresh System User token.
- 131030 → the recipient isn't in the dev-mode allowed list (App Dashboard > API Setup).
- Rate limits are handled automatically (backoff + `Retry-After`); don't add retry loops.

See `references/` for deeper guides: auth-and-config, waba-commands, output-and-filtering.
