## waba templates click-tracking

Toggle CTA URL click tracking for a template

```
waba templates click-tracking <template-id> [flags]
```

### Examples

```
  waba templates click-tracking 1234567890 --opt-out --category MARKETING
```

### Options

```
      --category string   the template's category (required by the API for this action)
  -h, --help              help for click-tracking
      --opt-out           opt the template OUT of click tracking (omit to opt back in)
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

