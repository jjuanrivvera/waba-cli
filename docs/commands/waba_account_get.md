## waba account get

Show the WABA node

```
waba account get [flags]
```

### Examples

```
  waba account get
  waba account get --fields id,name,health_status
  waba account get --fields disable_marketing_messages_on_cloud_api
```

### Options

```
      --fields string   comma-separated field projection (default "id,name,currency,timezone_id,message_template_namespace,account_review_status,business_verification_status,country,ownership_type")
  -h, --help            help for get
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

