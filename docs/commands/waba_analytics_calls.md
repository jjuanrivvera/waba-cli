## waba analytics calls

Calling API analytics

```
waba analytics calls [flags]
```

### Examples

```
  waba analytics calls --start 2026-08-01 --end 2026-08-17 --granularity DAILY --metric-types COUNT,COST
```

### Options

```
      --country-codes strings   filter to these ISO country codes
      --dimensions strings      breakdown dimensions (phone, direction, country)
      --directions strings      USER_INITIATED and/or BUSINESS_INITIATED
      --end string              window end: Unix timestamp or YYYY-MM-DD
      --granularity string      bucket size: HALF_HOUR|DAILY|MONTHLY (default "DAILY")
  -h, --help                    help for calls
      --metric-types strings    COUNT, COST, AVERAGE_DURATION
      --phone-numbers strings   filter to these phone numbers
      --start string            window start: Unix timestamp or YYYY-MM-DD
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

