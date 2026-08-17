## waba agent guard

Emit safety rules that block irreversible WhatsApp operations

### Synopsis

Generate safety configuration for an agent host driving this CLI.

Every runnable command is classified from the live tree: reads run freely, writes require
approval, and irreversible operations (delete/deregister/deprecate variants and the raw
'api' escape hatch, which can issue any method) are blocked outright.

The generated hook is the enforcement layer. Permission rules alone are literal prefixes, so
'./bin/waba templates delete', 'env X=1 waba templates delete' and quote-splitting all
walk straight past them; the hook matches the command position anywhere in the line.

Known limits, stated rather than papered over: variable indirection ($X delete) and shell
aliases or eval are not defeated. Running the agent in MCP-only mode is the hard guarantee;
the Bash hook is defence in depth.

```
waba agent guard [flags]
```

### Examples

```
waba agent guard --host claude-code
  waba agent guard --host claude-code --write
  waba agent guard --host codex
  waba agent guard --host opencode --all-writes
```

### Options

```
      --all-writes    block every write, not only the irreversible ones
      --dir string    directory to write into (default: the host's own location)
  -h, --help          help for guard
      --host string   agent host: claude-code, codex or opencode (default "claude-code")
      --write         write the configuration files instead of printing them
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

* [waba agent](waba_agent)	 - Generate agent-host safety configuration from this CLI's own command tree

