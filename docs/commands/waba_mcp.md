## waba mcp

MCP server management

### Synopsis

Manage MCP servers for AI assistants and code editors

### Options

```
  -h, --help   help for mcp
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
* [waba mcp claude](waba_mcp_claude)	 - Manage Claude Desktop MCP servers
* [waba mcp cursor](waba_mcp_cursor)	 - Manage Cursor MCP servers
* [waba mcp start](waba_mcp_start)	 - Start the MCP server
* [waba mcp stream](waba_mcp_stream)	 - Stream the MCP server over HTTP
* [waba mcp tools](waba_mcp_tools)	 - Export tools as JSON
* [waba mcp vscode](waba_mcp_vscode)	 - Manage VSCode MCP servers

