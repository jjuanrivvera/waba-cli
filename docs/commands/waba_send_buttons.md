## waba send buttons

Send up to 3 reply buttons

```
waba send buttons <body> [flags]
```

### Examples

```
  waba send buttons --to 573001112233 "Confirm your visit" --button yes:Sí --button no:No
```

### Options

```
      --button stringArray   reply button as id:title (repeatable, max 3)
      --footer string        optional footer text
      --header string        optional header text
  -h, --help                 help for buttons
      --reply-to string      wamid of the message this one replies to
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

