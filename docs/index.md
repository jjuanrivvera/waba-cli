# waba

**WhatsApp Cloud API from the command line.** Every message type, media, phone numbers,
message templates, the business profile, QR codes, WhatsApp Flows, Calling, groups,
analytics and account management — 102 of the 105 documented operations, generated from an
enumerated API spec and enforced by CI.

```console
$ waba send text --to 573001112233 "Your order shipped!"
MESSAGE_ID           WA_ID
wamid.HBgLNTcz...    573001112233
```

## Install

macOS / Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/jjuanrivvera/waba-cli/main/install.sh | sh
```

Homebrew: `brew install jjuanrivvera/waba-cli/waba-cli` · Scoop: `scoop install waba-cli`
(bucket `https://github.com/jjuanrivvera/scoop-waba-cli`) · Go:
`go install github.com/jjuanrivvera/waba-cli/cmd/waba@latest` — deb/rpm/apk on the
[releases page](https://github.com/jjuanrivvera/waba-cli/releases/latest).

## Set up

```sh
waba init      # token (hidden prompt) → verify → pick a phone number → done
waba doctor    # diagnose config, credentials, connectivity, token, WABA access
```

You need a [System User token](https://business.facebook.com/settings/system-users) with
`whatsapp_business_messaging` + `whatsapp_business_management`. Tokens live in the OS
keyring; headless machines use the encrypted-file fallback (`WABA_KEYRING_PASSWORD`).

## Everyday commands

```sh
waba send text     --to 573001112233 "Hola!"
waba send template --to 573001112233 --name order_update --lang es_MX --param "Juan"
waba send buttons  --to 573001112233 "Confirm?" --button yes:Sí --button no:No
waba messages read wamid.HBg...          # blue ticks
waba messages typing wamid.HBg...        # blue ticks + typing indicator

waba media upload ./catalogo.pdf
waba templates list --status APPROVED
waba phone list
waba analytics messaging --start 2026-08-01 --end 2026-08-17 --granularity DAY

waba api GET me -q fields=id,name        # raw escape hatch for anything else
```

Every list paginates (`--all`, `--limit`, `--after`), every output pipes
(`-o table|json|yaml|csv|id`, `--columns`, `--jq`), and every mutation supports
`--dry-run`, which prints the exact `curl` equivalent with the token redacted.

## Where next

- [AI agents](agents.md) — the MCP server and the generated safety guard.
- [Command reference](commands/waba.md) — every command, generated from the binary itself.
