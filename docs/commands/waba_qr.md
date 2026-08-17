## waba qr

Manage QR codes and short links (wa.me deep links)

### Options

```
  -h, --help   help for qr
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
* [waba qr create](waba_qr_create)	 - Create a QR code with a prefilled message
* [waba qr delete](waba_qr_delete)	 - Retire a QR code permanently
* [waba qr get](waba_qr_get)	 - Show one QR code
* [waba qr list](waba_qr_list)	 - List the number's QR codes
* [waba qr update](waba_qr_update)	 - Change a QR code's prefilled message

