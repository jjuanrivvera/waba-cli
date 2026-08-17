## waba templates delete-by-id

Delete one language variant by id

### Synopsis

The API requires BOTH the template id (hsm_id) and its name for a single-language
deletion.

```
waba templates delete-by-id <template-id> <name> [flags]
```

### Options

```
  -h, --help   help for delete-by-id
  -y, --yes    skip the confirmation prompt
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

* [waba templates](waba_templates)	 - Manage message templates

