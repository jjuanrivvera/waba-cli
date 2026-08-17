## waba send template

Send an approved message template

### Synopsis

Send a template by name and language. Simple body placeholders can be filled with
repeatable --param flags; anything richer (media headers, buttons) takes the documented
components array via --components.

```
waba send template [flags]
```

### Examples

```
  waba send template --to 573001112233 --name hello_world --lang en_US
  waba send template --to 573001112233 --name order_update --lang es_MX --param "Juan" --param "#1234"
  waba send template --to 573001112233 --name promo --lang es --components @components.json
```

### Options

```
      --components string   components array as JSON, @file, or @- (overrides --param)
  -h, --help                help for template
      --lang string         template language code (e.g. en_US, es_MX)
      --name string         template name
      --param stringArray   positional body parameter (repeatable, in order)
      --reply-to string     wamid of the message this one replies to
      --to string           recipient phone number in E.164 digits (e.g. 573001112233)
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

