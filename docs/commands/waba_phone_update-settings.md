## waba phone update-settings

Update the number's settings (calling, SIP, identity checks)

### Synopsis

Posts a settings document verbatim — the calling/SIP/voicemail shape is deeply nested
and Meta evolves it, so it is passed as JSON rather than flags. Note: enabling SIP turns
off the /calls endpoints and calling webhooks for the number.

```
waba phone update-settings [phone-number-id] [flags]
```

### Examples

```
  waba phone update-settings -d '{"calling":{"status":"ENABLED"}}'
  waba phone update-settings -d @calling-settings.json
```

### Options

```
  -d, --data string   settings document as JSON, @file, or @- for stdin
  -h, --help          help for update-settings
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

* [waba phone](waba_phone)	 - Manage business phone numbers

