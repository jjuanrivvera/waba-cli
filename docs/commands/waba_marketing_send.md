## waba marketing send

Send a marketing template via the Marketing Messages API

### Synopsis

Sends a MARKETING template through /marketing_messages, Meta's delivery-optimized
channel (formerly "MM Lite"). The WABA must be onboarded to the Marketing Messages API;
WABA-level toggles ride on `waba account update`.

```
waba marketing send [flags]
```

### Examples

```
  waba marketing send --to 573001112233 --name promo_agosto --lang es_MX --param "Juan"
  waba marketing send --to 573001112233 --name promo --lang es --components @components.json --fallback
```

### Options

```
      --components string   components array as JSON, @file, or @- (overrides --param)
      --fallback            product_policy CLOUD_API_FALLBACK instead of STRICT
  -h, --help                help for send
      --lang string         template language code
      --name string         template name
      --param stringArray   positional body parameter (repeatable, in order)
      --reply-to string     wamid of the message this one replies to
      --to string           recipient phone number in E.164 digits (e.g. 573001112233)
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

* [waba marketing](waba_marketing)	 - Marketing Messages API (requires MM API onboarding)

