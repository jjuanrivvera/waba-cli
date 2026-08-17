## waba phone register

Register the number for Cloud API messaging

### Synopsis

Registers the number so it can send messages, using the two-step verification PIN
(set one first with `waba phone set-pin` if the number has none). Rate limited by Meta to
10 attempts per number per 72 hours.

```
waba phone register [phone-number-id] [flags]
```

### Examples

```
  waba phone register --pin 123456
  waba phone register --pin 123456 --region DE
```

### Options

```
  -h, --help            help for register
      --pin string      6-digit two-step verification PIN
      --region string   data localization region (2-letter ISO country code)
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

* [waba phone](waba_phone)	 - Manage business phone numbers

