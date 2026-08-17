## waba apps subscribe

Subscribe the token's app to the WABA's webhooks

### Synopsis

Subscribes the app the access token belongs to. An alternate callback URL (with its
verify token) can override the app-level webhook for just this WABA.

```
waba apps subscribe [flags]
```

### Examples

```
  waba apps subscribe
  waba apps subscribe --callback-url https://bot.example.com/webhook --verify-token s3cret
```

### Options

```
      --callback-url string   override callback URL for this WABA
  -h, --help                  help for subscribe
      --verify-token string   verify token for the override callback
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

* [waba apps](waba_apps)	 - Webhook subscriptions (subscribed apps)

