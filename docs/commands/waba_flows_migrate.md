## waba flows migrate

Copy flows from another WABA

### Synopsis

Copies (never moves) flows between WABAs owned by the same Meta business. Name
collisions are skipped per flow; published state is preserved; new ids are issued.

```
waba flows migrate --from <source-waba-id> [flags]
```

### Examples

```
  waba flows migrate --from 111222333
  waba flows migrate --from 111222333 --names lead_gen,booking
```

### Options

```
      --from string     source WABA id
  -h, --help            help for migrate
      --names strings   only migrate these flows (default: all)
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

