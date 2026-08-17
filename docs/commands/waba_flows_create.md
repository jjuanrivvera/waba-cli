## waba flows create

Create a flow

```
waba flows create <name> [flags]
```

### Examples

```
  waba flows create "Agendar cita" --categories APPOINTMENT_BOOKING
  waba flows create "Encuesta" --categories SURVEY --json flow.json --publish
```

### Options

```
      --categories strings    flow categories (e.g. SIGN_UP, APPOINTMENT_BOOKING, SURVEY, OTHER)
      --clone-from string     flow id to clone
      --endpoint-uri string   data-exchange endpoint URI
  -h, --help                  help for create
      --json string           path to a flow.json to attach on creation
      --publish               publish immediately if the flow validates
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

* [waba flows](waba_flows)	 - Manage WhatsApp Flows

