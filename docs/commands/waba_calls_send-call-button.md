## waba calls send-call-button

Send a voice-call button message

### Synopsis

Sends an interactive voice_call button the user can tap to call the business.

```
waba calls send-call-button <user-number> [flags]
```

### Examples

```
  waba calls send-call-button 573001112233 --body "¿Prefieres hablar?" --display-text "Llámanos" --ttl-minutes 1440
```

### Options

```
      --body string           message text above the button
      --display-text string   button label (≤20 chars) (default "Call")
  -h, --help                  help for send-call-button
      --payload string        opaque payload echoed in call webhooks (≤512 chars)
      --ttl-minutes int       how long the button stays tappable (1–43200)
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

