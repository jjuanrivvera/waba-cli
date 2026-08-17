## waba auth login

Store an access token and verify it against the API

```
waba auth login [flags]
```

### Examples

```
# Interactive (the token prompt does not echo)
  waba auth login

  # Scripted
  waba auth login --token "$MY_TOKEN"
```

### Options

```
  -h, --help           help for login
      --token string   access token (omit to be prompted without echo)
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

* [waba auth](waba_auth)	 - Log in, log out, and inspect authentication

