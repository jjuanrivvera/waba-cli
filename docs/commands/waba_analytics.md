## waba analytics

Messaging, conversation, pricing, template and call analytics

### Options

```
  -h, --help   help for analytics
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
* [waba analytics calls](waba_analytics_calls)	 - Calling API analytics
* [waba analytics conversations](waba_analytics_conversations)	 - Conversation counts and costs
* [waba analytics groups](waba_analytics_groups)	 - WhatsApp groups analytics
* [waba analytics messaging](waba_analytics_messaging)	 - Sent and delivered message counts
* [waba analytics pricing](waba_analytics_pricing)	 - Per-message pricing analytics
* [waba analytics template-groups](waba_analytics_template-groups)	 - Template group analytics
* [waba analytics templates](waba_analytics_templates)	 - Per-template analytics (sent, delivered, read, clicked)

