## waba messages typing

Mark read and show a typing indicator

### Synopsis

Combines the read receipt with a typing indicator, which stays visible for up to 25
seconds or until the next message is sent.

```
waba messages typing <wamid> [flags]
```

### Examples

```
  waba messages typing wamid.HBgLNTczMDA...
```

### Options

```
  -h, --help   help for typing
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

* [waba messages](waba_messages)	 - Mark messages read and show typing indicators

