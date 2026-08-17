## waba phone

Manage business phone numbers

### Options

```
  -h, --help   help for phone
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
* [waba phone deregister](waba_phone_deregister)	 - Deregister the number from Cloud API
* [waba phone get](waba_phone_get)	 - Show one phone number
* [waba phone list](waba_phone_list)	 - List the WABA's phone numbers
* [waba phone name-status](waba_phone_name-status)	 - Check the display name review status
* [waba phone register](waba_phone_register)	 - Register the number for Cloud API messaging
* [waba phone request-code](waba_phone_request-code)	 - Request an ownership verification code
* [waba phone set-pin](waba_phone_set-pin)	 - Set or change the two-step verification PIN
* [waba phone settings](waba_phone_settings)	 - Show the number's settings (incl. calling and SIP)
* [waba phone update-settings](waba_phone_update-settings)	 - Update the number's settings (calling, SIP, identity checks)
* [waba phone verify-code](waba_phone_verify-code)	 - Submit the received verification code

