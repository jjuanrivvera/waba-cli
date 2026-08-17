# Command deep-dive

## Sending messages

All sends go to the account's default phone number (`--phone-id` overrides). The recipient
is E.164 digits without `+` (e.g. `573001112233`). Every send accepts `--reply-to <wamid>`
to thread a reply.

```sh
waba send text --to 573001112233 "Hola" --preview-url
waba send image --to 573001112233 --link https://... --caption "..."   # or --id <media-id>
waba send document --to 573001112233 --id 1013859 --filename factura.pdf
waba send location --to 573001112233 --lat 4.71 --lng -74.07 --name "Oficina"
waba send contacts --to 573001112233 -d @contacts.json
waba send reaction --to 573001112233 --message-id wamid.X --emoji 👍   # empty --emoji removes
```

**Templates** (the only way to reach users outside the 24-hour window):

```sh
waba send template --to <num> --name hello_world --lang en_US
waba send template --to <num> --name order_update --lang es_MX --param "Juan" --param "#42"
waba send template --to <num> --name promo --lang es --components @components.json
```

`--param` fills positional `{{1}}`-style body placeholders in order; media headers and
buttons need the full documented components array via `--components`.

**Interactive**: `buttons` (≤3 reply buttons `id:title`), `list` (≤10 rows
`id:title[:description]`), `cta-url`, `flow` (`--flow-id|--flow-name`, `--flow-cta`,
`--mode draft` to test unpublished flows). `send interactive -d @payload.json` passes any
interactive object verbatim (address messages, product lists).

## Media

Uploaded media lives 30 days; a `media url` link expires in 5 minutes (which is why
`media download` fetches immediately). Downloads carry the bearer token, restricted to
Meta's CDN hosts.

## Templates

- `templates list --status APPROVED|PENDING|REJECTED --name <substr> --language <code>`
- `templates create -d @tpl.json` — goes to Meta review; ≤100 creations/WABA/hour.
- `templates delete <name>` removes **every language variant**; `delete-by-id <id> <name>`
  removes one; `bulk-delete` takes ≤100 ids and is all-or-nothing.
- `templates compare <id> <other> --days 7|30|60|90` needs ≥1,000 sends each.

## Analytics

Granularity enums differ per edge — the CLI validates them:

| Command | Granularity | Notes |
|---|---|---|
| `analytics messaging` | `HALF_HOUR\|DAY\|MONTH` | sent/delivered counts |
| `analytics conversations` | `HALF_HOUR\|DAILY\|MONTHLY` | cost + count, `--dimensions` |
| `analytics pricing` | `HALF_HOUR\|DAILY\|MONTHLY` | per-message pricing model |
| `analytics calls` | `HALF_HOUR\|DAILY\|MONTHLY` | Calling API |
| `analytics templates` | `DAILY` | needs `waba account enable-insights` (irreversible) |

`--start`/`--end` take Unix timestamps or `YYYY-MM-DD`.

## Flows

```sh
waba flows create "Agendar" --categories APPOINTMENT_BOOKING --json flow.json
waba flows upload-json <flow-id> flow.json     # validate + attach
waba flows preview <flow-id>                   # 30-day embeddable preview URL
waba flows publish <flow-id>                   # DRAFT → PUBLISHED (deprecate to retire)
waba flows migrate --from <source-waba-id>     # copy flows between WABAs (same business)
```

## Calling

`calls connect|pre-accept|accept|reject|terminate` drive the WebRTC signalling
(`--sdp` carries the RFC 8866 session description); `calls permissions <num>` checks the
user's permission state; `calls request-permission` / `calls send-call-button` send the
interactive prompts. Calling settings (including SIP) live on
`phone settings` / `phone update-settings` — note that enabling SIP disables the `/calls`
endpoints for that number.

## Groups

Requires an Official Business Account. ≤8 participants, joined via invite link only.
`groups send|pin|unpin` cover group messaging; join-request moderation is not wrapped
(see `waba api`).

## The escape hatch

```sh
waba api GET  <waba-id>/message_template_previews
waba api POST <group-id>/join_requests -d '{...}'
```

Any documented Graph edge, with the account's token, version prefix, `--dry-run` and
output formatting.
