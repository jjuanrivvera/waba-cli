## waba templates

Manage message templates

### Options

```
  -h, --help   help for templates
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
* [waba templates bulk-delete](waba_templates_bulk-delete)	 - Delete up to 100 templates by id
* [waba templates click-tracking](waba_templates_click-tracking)	 - Toggle CTA URL click tracking for a template
* [waba templates compare](waba_templates_compare)	 - Compare two templates' performance
* [waba templates create](waba_templates_create)	 - Create a template (goes to Meta review)
* [waba templates delete](waba_templates_delete)	 - Delete a template by name — ALL its languages
* [waba templates delete-by-id](waba_templates_delete-by-id)	 - Delete one language variant by id
* [waba templates edit](waba_templates_edit)	 - Edit a template (re-triggers review)
* [waba templates get](waba_templates_get)	 - Show one template (components included)
* [waba templates list](waba_templates_list)	 - List the WABA's templates

