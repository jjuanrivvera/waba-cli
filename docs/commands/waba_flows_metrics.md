## waba flows metrics

Endpoint metrics for a published flow

### Synopsis

Meta has announced this Metrics API will be discontinued on 2026-04-30; it still
works today for published flows with an endpoint and ≥250 requests in the window.

```
waba flows metrics <flow-id> [flags]
```

### Examples

```
  waba flows metrics 123456 --metric ENDPOINT_REQUEST_COUNT --since 2026-08-01 --until 2026-08-17
```

### Options

```
      --granularity string   DAY|HOUR|LIFETIME (default "DAY")
  -h, --help                 help for metrics
      --metric string        ENDPOINT_REQUEST_COUNT|ENDPOINT_REQUEST_ERROR|ENDPOINT_REQUEST_ERROR_RATE|ENDPOINT_REQUEST_LATENCY_SECONDS_CEIL|ENDPOINT_AVAILABILITY (default "ENDPOINT_REQUEST_COUNT")
      --since string         window start (YYYY-MM-DD)
      --until string         window end (YYYY-MM-DD)
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

* [waba flows](waba_flows)	 - Manage WhatsApp Flows

