## waba init

First-run wizard: token, WABA id, default phone number

### Synopsis

Set up an account interactively (or fully via flags for scripts). The wizard verifies the
token, stores it in the OS keyring, discovers the WABA's phone numbers so you can pick the
default sender, and smoke-tests the result.

```
waba init [flags]
```

### Examples

```
waba init
  waba init --name prod --token "$TOKEN" --waba-id 102290... --phone-number-id 106540...
```

### Options

```
      --app-id string            Meta app id (needed only for resumable uploads)
      --business-id string       Meta business portfolio id (optional)
  -h, --help                     help for init
      --name string              account name (prompted if omitted)
      --phone-number-id string   default business phone number id
      --token string             access token (prompted without echo if omitted)
      --waba-id string           WhatsApp Business Account id
```

### Options inherited from parent commands

```
      --account string    named account to use
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
```

### SEE ALSO

* [waba](waba)	 - WhatsApp Cloud API from the command line

