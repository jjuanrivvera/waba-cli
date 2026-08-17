## waba block add

Block users (max 1,000 per call)

### Synopsis

Only users who messaged the business in the last 24 hours can be blocked; the
blocklist holds at most 64,000 entries.

```
waba block add <number>... [flags]
```

### Examples

```
  waba block add 573001112233 573004445566
```

### Options

```
  -h, --help   help for add
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

* [waba block](waba_block)	 - Manage the blocklist

