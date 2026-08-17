## waba analytics conversations

Conversation counts and costs

```
waba analytics conversations [flags]
```

### Examples

```
  waba analytics conversations --start 2026-08-01 --end 2026-08-17 --granularity DAILY \
    --dimensions CONVERSATION_CATEGORY,COUNTRY
```

### Options

```
      --conversation-categories strings   AUTHENTICATION, MARKETING, SERVICE, UTILITY
      --conversation-directions strings   BUSINESS_INITIATED, USER_INITIATED
      --conversation-types strings        FREE_ENTRY_POINT, FREE_TIER, REGULAR
      --dimensions strings                breakdown dimensions (e.g. CONVERSATION_CATEGORY, COUNTRY, PHONE)
      --end string                        window end: Unix timestamp or YYYY-MM-DD
      --granularity string                bucket size: HALF_HOUR|DAILY|MONTHLY (default "DAILY")
  -h, --help                              help for conversations
      --metric-types strings              COST and/or CONVERSATION
      --phone-numbers strings             filter to these phone numbers
      --start string                      window start: Unix timestamp or YYYY-MM-DD
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

