## waba completion

Generate a shell completion script

### Synopsis

Generate a shell completion script.

Completion is worth installing here: it completes site names, output formats, column names
plus resource and verb names.

  bash:  waba completion bash > /etc/bash_completion.d/waba
  zsh:   waba completion zsh > "${fpath[1]}/_waba"
  fish:  waba completion fish > ~/.config/fish/completions/waba.fish
  pwsh:  waba completion powershell | Out-String | Invoke-Expression

```
waba completion [bash|zsh|fish|powershell] [flags]
```

### Options

```
  -h, --help   help for completion
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

