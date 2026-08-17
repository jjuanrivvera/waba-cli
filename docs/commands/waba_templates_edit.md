## waba templates edit

Edit a template (re-triggers review)

### Synopsis

Only APPROVED, REJECTED or PAUSED templates can be edited; approved templates allow
10 edits per 30 days (1 per 24h).

```
waba templates edit <template-id> [flags]
```

### Examples

```
  waba templates edit 1234567890 -d '{"components":[...]}'
```

### Options

```
  -d, --data string   fields to change as JSON, @file, or @- for stdin
  -h, --help          help for edit
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

