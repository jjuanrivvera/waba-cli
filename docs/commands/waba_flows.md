## waba flows

Manage WhatsApp Flows

### Options

```
  -h, --help   help for flows
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
* [waba flows assets](waba_flows_assets)	 - List the flow's assets
* [waba flows create](waba_flows_create)	 - Create a flow
* [waba flows delete](waba_flows_delete)	 - Delete a DRAFT flow
* [waba flows deprecate](waba_flows_deprecate)	 - Deprecate a published flow
* [waba flows get](waba_flows_get)	 - Show one flow
* [waba flows list](waba_flows_list)	 - List the WABA's flows
* [waba flows metrics](waba_flows_metrics)	 - Endpoint metrics for a published flow
* [waba flows migrate](waba_flows_migrate)	 - Copy flows from another WABA
* [waba flows preview](waba_flows_preview)	 - Get an embeddable preview URL (valid 30 days)
* [waba flows publish](waba_flows_publish)	 - Publish a draft flow
* [waba flows update](waba_flows_update)	 - Update flow metadata (name, categories, endpoint)
* [waba flows upload-json](waba_flows_upload-json)	 - Upload the flow's JSON definition

