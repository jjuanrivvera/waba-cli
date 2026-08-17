## waba calls

WhatsApp Calling: place, answer and manage calls

### Options

```
  -h, --help   help for calls
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
* [waba calls accept](waba_calls_accept)	 - Accept an inbound call
* [waba calls connect](waba_calls_connect)	 - Start a business-initiated call
* [waba calls permissions](waba_calls_permissions)	 - Check the call permission state for a user
* [waba calls pre-accept](waba_calls_pre-accept)	 - Pre-accept an inbound call (early media setup)
* [waba calls reject](waba_calls_reject)	 - Reject an inbound call
* [waba calls request-permission](waba_calls_request-permission)	 - Send an interactive call-permission request
* [waba calls send-call-button](waba_calls_send-call-button)	 - Send a voice-call button message
* [waba calls terminate](waba_calls_terminate)	 - End an active call

