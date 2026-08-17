## waba groups create

Create a group

### Synopsis

Groups hold at most 8 participants and 10,000 groups per business number. Users join
via the invite link — there is no API to add participants directly.

```
waba groups create <subject> [flags]
```

### Examples

```
  waba groups create "Clientes VIP" --description "Ofertas exclusivas"
```

### Options

```
      --description string   group description
  -h, --help                 help for create
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

* [waba groups](waba_groups)	 - WhatsApp groups (requires an Official Business Account)

