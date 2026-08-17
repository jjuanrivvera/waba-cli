## waba api

Raw authenticated Graph API request

### Synopsis

Perform an arbitrary Graph API request with the account's token and Graph version.
PATH is version-relative ("me", "{waba-id}/flows"). Honors --dry-run and -o.

```
waba api <METHOD> <PATH> [flags]
```

### Examples

```
waba api GET me -q fields=id,name
  waba api GET 102290129340398/phone_numbers
  waba api POST 106540352242922/messages -d '{"messaging_product":"whatsapp","to":"...","type":"text","text":{"body":"hi"}}'
  waba api DELETE 102290129340398/message_templates -q name=old_template
```

### Options

```
  -d, --data string          JSON request body, @file, or @- for stdin
  -H, --header stringArray   extra header key:value (repeatable)
  -h, --help                 help for api
  -q, --query stringArray    query parameter key=value (repeatable)
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

* [waba](waba)	 - WhatsApp Cloud API from the command line

