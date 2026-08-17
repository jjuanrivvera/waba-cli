## waba calls request-permission

Send an interactive call-permission request

### Synopsis

Free-form call permission requests only work inside an open customer-service window;
outside it, send a template containing a call_permission_request component. Meta enforces
1 request per user per 24h and 2 per 7 days.

```
waba calls request-permission <user-number> [flags]
```

### Examples

```
  waba calls request-permission 573001112233 --body "¿Podemos llamarte para coordinar la visita?"
```

### Options

```
      --body string   message text shown with the permission request
  -h, --help          help for request-permission
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

* [waba calls](waba_calls)	 - WhatsApp Calling: place, answer and manage calls

