## waba send flow

Send a WhatsApp Flow

```
waba send flow <body> [flags]
```

### Examples

```
  waba send flow --to 573001112233 "Book your appointment" --flow-id 123456 --flow-cta "Agendar" --flow-token tok-1
```

### Options

```
      --flow-action string           flow action: navigate|data_exchange (default "navigate")
      --flow-action-payload string   action payload JSON (screen + data) for navigate
      --flow-cta string              button label that opens the flow
      --flow-id string               flow id (or use --flow-name)
      --flow-name string             flow name (or use --flow-id)
      --flow-token string            opaque token echoed back in the flow webhook
      --footer string                optional footer text
      --header string                optional header text
  -h, --help                         help for flow
      --mode string                  flow mode: draft to test an unpublished flow
      --reply-to string              wamid of the message this one replies to
      --to string                    recipient phone number in E.164 digits (e.g. 573001112233)
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

