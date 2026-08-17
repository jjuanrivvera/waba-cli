## waba account update

Update WABA-level settings

### Synopsis

Posts WABA node parameters verbatim — used for Marketing Messages toggles and other
account-level flags Meta adds over time.

```
waba account update [flags]
```

### Examples

```
  waba account update -d '{"disable_marketing_messages_on_cloud_api":true}'
```

### Options

```
  -d, --data string   parameters as JSON, @file, or @- for stdin
  -h, --help          help for update
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

* [waba account](waba_account)	 - Inspect and update the WhatsApp Business Account (WABA)

