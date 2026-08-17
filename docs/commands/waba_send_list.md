## waba send list

Send an interactive list menu

```
waba send list <body> [flags]
```

### Examples

```
  waba send list --to 573001112233 "Pick a service" --button-text Menú \
    --section "Servicios" --row "rev:Revisión:Diagnóstico completo" --row "rep:Reparación"
```

### Options

```
      --button-text string   text on the button that opens the list (default "Menu")
      --footer string        optional footer text
      --header string        optional header text
  -h, --help                 help for list
      --reply-to string      wamid of the message this one replies to
      --row stringArray      row as id:title[:description] (repeatable, max 10)
      --section string       section title for the rows
      --to string            recipient phone number in E.164 digits (e.g. 573001112233)
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

* [waba send](waba_send)	 - Send WhatsApp messages

