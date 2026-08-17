## waba

WhatsApp Cloud API from the command line

### Synopsis

waba is a command-line client for Meta's WhatsApp Cloud API: send every message type,
manage media, phone numbers, message templates, the business profile, QR codes, WhatsApp
Flows, calling, groups and analytics — 102 documented operations, enumerated in
api-manifest.json.

Profiles ("accounts") bundle a WABA id, a default phone number id and an app id, so daily
use is just 'waba send text --to ... "hi"'. Tokens live in the OS keyring.

### Examples

```
  # First-run setup: token, WABA id, default phone number
  waba init

  # Send messages
  waba send text --to 573001112233 "Your order shipped!"
  waba send template --to 573001112233 --name order_update --lang es_MX
  waba send image --to 573001112233 --link https://example.com/cat.jpg

  # Templates
  waba templates list --status APPROVED
  waba templates create --data @welcome.json

  # Inspect the account
  waba phone list
  waba analytics messaging --start 2026-08-01 --end 2026-08-17 --granularity DAY

  # Anything else in the Graph API
  waba api GET me
```

### Options

```
      --account string    named account to use
      --app-id string     Meta app id (overrides the account default)
      --base-url string   override the Graph API base URL
      --columns strings   columns to show in table/csv output
      --dry-run           print the equivalent curl command and send nothing
  -h, --help              help for waba
      --jq string         filter the result through a gojq expression
      --no-color          disable colored output
  -o, --output string     output format: table|json|yaml|csv|id (default "table")
      --phone-id string   business phone number id (overrides the account default)
      --quiet             suppress notes and warnings
      --rps float         client-side request rate limit (requests/second) (default 10)
      --show-token        do not redact credentials in --dry-run output
      --timeout int       per-request timeout in seconds (default 60)
  -v, --verbose           trace requests to stderr
      --waba-id string    WhatsApp Business Account id (overrides the account default)
```

### SEE ALSO

* [waba account](waba_account)	 - Inspect and update the WhatsApp Business Account (WABA)
* [waba agent](waba_agent)	 - Generate agent-host safety configuration from this CLI's own command tree
* [waba alias](waba_alias)	 - Define shortcuts for longer commands
* [waba analytics](waba_analytics)	 - Messaging, conversation, pricing, template and call analytics
* [waba api](waba_api)	 - Raw authenticated Graph API request
* [waba apps](waba_apps)	 - Webhook subscriptions (subscribed apps)
* [waba auth](waba_auth)	 - Log in, log out, and inspect authentication
* [waba automation](waba_automation)	 - Conversational components: commands and ice breakers
* [waba block](waba_block)	 - Manage the blocklist
* [waba calls](waba_calls)	 - WhatsApp Calling: place, answer and manage calls
* [waba commerce](waba_commerce)	 - Cart and catalog visibility settings
* [waba completion](waba_completion)	 - Generate a shell completion script
* [waba config](waba_config)	 - Inspect and edit the configuration
* [waba doctor](waba_doctor)	 - Diagnose configuration, credentials and connectivity
* [waba flows](waba_flows)	 - Manage WhatsApp Flows
* [waba groups](waba_groups)	 - WhatsApp groups (requires an Official Business Account)
* [waba init](waba_init)	 - First-run wizard: token, WABA id, default phone number
* [waba marketing](waba_marketing)	 - Marketing Messages API (requires MM API onboarding)
* [waba mcp](waba_mcp)	 - MCP server management
* [waba media](waba_media)	 - Upload, inspect, download and delete media
* [waba messages](waba_messages)	 - Mark messages read and show typing indicators
* [waba phone](waba_phone)	 - Manage business phone numbers
* [waba profile](waba_profile)	 - Manage the WhatsApp Business profile
* [waba qr](waba_qr)	 - Manage QR codes and short links (wa.me deep links)
* [waba send](waba_send)	 - Send WhatsApp messages
* [waba templates](waba_templates)	 - Manage message templates
* [waba update](waba_update)	 - Update waba to the latest release
* [waba uploads](waba_uploads)	 - Resumable uploads (template header media, profile pictures)
* [waba version](waba_version)	 - Print version information

