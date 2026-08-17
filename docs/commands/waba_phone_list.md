## waba phone list

List the WABA's phone numbers

```
waba phone list [flags]
```

### Examples

```
  waba phone list
  waba phone list --fields id,display_phone_number,status,quality_rating
```

### Options

```
      --after string    continue from a pagination cursor
      --all             fetch every page
      --before string   page backwards from a cursor
      --fields string   comma-separated field projection
  -h, --help            help for list
      --limit int       items per page
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

