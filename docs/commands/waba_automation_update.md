## waba automation update

Configure commands and ice-breaker prompts

### Synopsis

Commands appear when the user types “/”; ice breakers are tappable prompts on the
first contact. Emojis and markdown are not supported in either.

```
waba automation update [flags]
```

### Examples

```
  waba automation update \
    --command "cotizar:Recibe una cotización" --command "horario:Horario de atención" \
    --prompt "¿Cuánto cuesta una revisión?" --prompt "¿Atienden hoy?"
```

### Options

```
      --command stringArray   bot command as name:description (repeatable, max 30)
  -h, --help                  help for update
      --prompt stringArray    ice-breaker prompt text (repeatable, max 4, ≤80 chars)
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

* [waba automation](waba_automation)	 - Conversational components: commands and ice breakers

