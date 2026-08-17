## waba auth

Log in, log out, and inspect authentication

### Synopsis

Manage the Graph API access token for an account. The token is stored in the OS keyring
(or the encrypted-file fallback on headless machines — set WABA_KEYRING_PASSWORD).

In practice the right credential is a System User token generated in Meta Business
Manager with the whatsapp_business_messaging and whatsapp_business_management
permissions; App Dashboard temporary tokens work but expire within 24 hours.

### Options

```
  -h, --help   help for auth
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
* [waba auth login](waba_auth_login)	 - Store an access token and verify it against the API
* [waba auth logout](waba_auth_logout)	 - Remove the stored token for the active account
* [waba auth status](waba_auth_status)	 - Show the active account, token backend and identity

