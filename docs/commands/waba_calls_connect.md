## waba calls connect

Start a business-initiated call

### Synopsis

Requires an existing call permission from the user (`waba calls permissions`),
calling enabled on the number, and a calls webhook. Limited by Meta to 10,000 call
initiations per number per 24h; unavailable for businesses in some countries.

```
waba calls connect [flags]
```

### Examples

```
  waba calls connect --to 573001112233 --sdp "$(cat offer.sdp)"
```

### Options

```
      --callback-data string   opaque data echoed in call webhooks (≤512 chars)
  -h, --help                   help for connect
      --sdp string             RFC 8866 session description (inline or @file via shell)
      --sdp-type string        sdp_type: offer or answer (default "offer")
      --to string              callee phone number
```

### Options inherited from parent commands

```
      --account string    named account to use
      --app-id string     Meta app id (overrides the account default)
      --base-url string   override the Graph API base URL
      --columns strings   columns to show in table/csv output
      --dry-run           print the equivalent curl command and send nothing
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

* [waba calls](waba_calls)	 - WhatsApp Calling: place, answer and manage calls

