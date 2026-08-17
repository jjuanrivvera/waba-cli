## waba analytics template-groups

Template group analytics

```
waba analytics template-groups [flags]
```

### Options

```
      --end string             window end: Unix timestamp or YYYY-MM-DD
      --granularity string     bucket size: DAILY (default "DAILY")
      --group-ids strings      template group ids to report on
  -h, --help                   help for template-groups
      --metric-types strings   cost, clicked, delivered, read, sent
      --start string           window start: Unix timestamp or YYYY-MM-DD
      --waba-timezone          bucket days in the WABA's timezone instead of UTC
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

* [waba analytics](waba_analytics)	 - Messaging, conversation, pricing, template and call analytics

