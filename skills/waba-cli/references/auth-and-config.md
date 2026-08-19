# Auth and configuration

## The token

The Cloud API authorizes every call with a bearer access token. For anything durable use a
**System User token** from Meta Business Manager (Business Settings → Users → System users):
create a system user, assign the WABA and app with full control, then generate a token with
`whatsapp_business_messaging` and `whatsapp_business_management`. App Dashboard temporary
tokens expire within 24 hours — fine for a quick test, wrong for automation.

```sh
waba init                          # wizard: token (hidden prompt) → verify → pick phone number
waba auth login --token "$TOKEN"   # replace the token for the active account
waba auth status                   # identity, backend, validity (alias: waba auth whoami)
waba auth logout
```

Tokens are stored in the OS keyring (macOS Keychain, Linux Secret Service, Windows
Credential Manager). Headless Linux boxes without a Secret Service fall back automatically
to an AES-256-GCM encrypted file keyed by a per-machine secret — `waba init` works on a
fresh CI runner or container with zero setup. Setting `WABA_KEYRING_PASSWORD` (or writing
it to `<config-dir>/keyring-password`, chmod 600) upgrades that fallback to real password
encryption; a password file also makes the choice persistent for cron/ssh sessions.
`WABA_KEYRING_BACKEND=file|keyring` forces a backend.

## Accounts (profiles)

An account bundles everything one business needs:

| Field | Meaning |
|---|---|
| `waba_id` | WhatsApp Business Account id (templates, analytics, flows hang off it) |
| `phone_number_id` | default sender — an id from `waba phone list`, not the display number |
| `app_id` | Meta app id, needed only for resumable uploads |
| `business_id` | Meta business portfolio id (optional) |
| `graph_version` | pinned Graph API version (default v25.0) |

```sh
waba config set phone_number_id 106540352242922
waba config list-accounts
waba config use staging
waba phone list --account prod        # one-off account selection (--profile also works)
```

## Environment overrides

Everything can come from the environment (CI, containers — no config file needed):
`WABA_ACCESS_TOKEN`, `WABA_ACCOUNT`, `WABA_WABA_ID`, `WABA_PHONE_NUMBER_ID`,
`WABA_APP_ID`, `WABA_BUSINESS_ID`, `WABA_GRAPH_VERSION`, `WABA_BASE_URL`.

Precedence everywhere: **flag > environment > config file > default**.
