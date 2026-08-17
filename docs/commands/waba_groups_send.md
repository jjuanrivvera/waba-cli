## waba groups send

Send a text message to a group

### Synopsis

Groups accept text, media and text/media templates — interactive, authentication and
commerce messages are not supported. For media or templates, use `waba send … --to
<group-id>` with recipient_type group via `waba api`.

```
waba groups send <group-id> <text> [flags]
```

### Examples

```
  waba groups send 120363043211234567@g.us "La oferta termina hoy"
```

### Options

```
  -h, --help   help for send
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

