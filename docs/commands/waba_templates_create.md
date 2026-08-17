## waba templates create

Create a template (goes to Meta review)

### Synopsis

Creates a template from a JSON document — the components array (header, body,
footer, buttons) is too rich for flags. Meta reviews new templates before they can be
sent; creation is limited to 100 templates per WABA per hour.

```
waba templates create [flags]
```

### Examples

```
  waba templates create -d '{"name":"order_update","language":"es_MX","category":"UTILITY","components":[{"type":"BODY","text":"Hola {{1}}, tu pedido {{2}} va en camino."}]}'
  waba templates create -d @welcome.json
```

### Options

```
  -d, --data string   template document as JSON, @file, or @- for stdin
  -h, --help          help for create
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

