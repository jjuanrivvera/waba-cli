## waba profile update

Update business profile fields

### Synopsis

Updates only the fields you pass. The profile picture takes a handle from the
resumable upload API (`waba uploads start` + `waba uploads upload`), not a URL.

```
waba profile update [flags]
```

### Examples

```
  waba profile update --about "Reparación de neveras en Bogotá" --vertical OTHER
  waba profile update --website https://rivera-refrigeracion.com --email hola@example.com
```

### Options

```
      --about string            the about text (max 139 chars)
      --address string          business address
      --description string      business description
      --email string            contact email
  -h, --help                    help for update
      --picture-handle string   profile picture handle from the resumable upload API
      --vertical string         industry vertical (e.g. OTHER, RETAIL, PROF_SERVICES)
      --website stringArray     website URL (repeatable, max 2)
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

* [waba profile](waba_profile)	 - Manage the WhatsApp Business profile

