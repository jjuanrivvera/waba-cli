# AI agents

`waba` is built to be driven by AI agents as safely as by humans: every command carries a
read-only / write / destructive annotation, and two commands turn those annotations into
infrastructure.

## MCP server

```sh
waba mcp claude   # install into Claude Desktop
waba mcp cursor   # Cursor
waba mcp vscode   # VS Code
waba mcp start    # or run the stdio server directly
```

The server exposes the command tree as MCP tools (`waba_send_text`,
`waba_templates_list`, …) that replay the real cobra commands — same client, same keyring,
same account profiles, same `--dry-run`. Hosts see each tool's `readOnlyHint` /
`destructiveHint` and can gate writes accordingly.

Deliberately **not** exposed: `auth`, `config`, `init`, `alias`, `update`, `doctor`,
`agent`, and the flags that would let an agent retarget the server or read the credential
(`--account`/`--profile`, `--base-url`, `--waba-id`, `--phone-id`, `--app-id`,
`--show-token`). The server operates as the account that was active at startup, and an
agent cannot switch it.

## Agent guard

```sh
waba agent guard --host claude-code --write   # .claude/settings.json + PreToolUse hook
waba agent guard --host codex --write         # .codex/config.toml (read-only sandbox)
waba agent guard --host opencode --write      # opencode.json permissions
waba agent classify                            # see how every command is classified
```

The guard classifies every command from the live tree and emits host safety config:

- **Blocked outright** — irreversible operations: every `delete` variant,
  `phone deregister`, `flows deprecate`, `apps unsubscribe`,
  `groups remove-participants`, mutating methods through the raw `api` command, and
  `alias set` (an alias could repoint a safe-looking name at a destructive command).
- **Approval required** — ordinary writes: sends, template creation, profile updates.
- **Free** — reads.

For Claude Code the enforcement layer is a generated `PreToolUse` hook that matches the
command position anywhere in the line — path-prefixed binaries, `env`-prefixed
invocations, quote-splitting (`de""lete`), chained commands and alias spellings are all
denied; near-miss tool names and blocked verbs inside arguments stay allowed. The
permission rules in `settings.json` are belt-and-suspenders on top.

Known limits, stated rather than papered over: variable indirection (`$X delete`) and
shell `eval` are not defeated — running the agent MCP-only is the hard guarantee; the Bash
hook is defence in depth.

## Recommended setup for unattended agents

```sh
waba agent guard --host claude-code --all-writes --write   # block every write, not just irreversible ones
```

Then let the agent read freely (`waba templates list`, `waba analytics …`, `waba phone
list`) and require a human for anything that would message a customer.
