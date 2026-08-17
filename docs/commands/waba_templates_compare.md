## waba templates compare

Compare two templates' performance

### Synopsis

Both templates must belong to the same WABA and have been sent ≥1,000 times in the
window. The lookback is exactly 7, 30, 60 or 90 days ending now.

```
waba templates compare <template-id> <other-template-id> [flags]
```

### Examples

```
  waba templates compare 111 222 --days 30
```

### Options

```
      --days int   lookback window: 7, 30, 60 or 90 days (default 7)
  -h, --help       help for compare
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

* [waba templates](waba_templates)	 - Manage message templates

