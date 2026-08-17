# Security Policy

## Supported versions

The latest released minor version receives security fixes.

| Version | Supported |
|---|---|
| 0.1.x | ✅ |

## Reporting a vulnerability

Report privately through
[GitHub Security Advisories](https://github.com/jjuanrivvera/waba-cli/security/advisories/new).
Please do not open a public issue for a vulnerability.

Expect an acknowledgement within 72 hours and an assessment within a week.

## How this tool handles your credentials

- **Credentials are never written to the config file.** They go to the OS keyring (macOS
  Keychain, Linux Secret Service, Windows Credential Manager). The config holds only
  non-secret settings: site URLs, your account email, the chosen auth method, the OAuth
  client id and cloud id.
- **The encrypted-file fallback** (`ATLASSIAN_KEYRING_PASSWORD`, for headless hosts) uses
  AES-256-GCM with a PBKDF2-HMAC-SHA256 key at 600,000 iterations and a fresh random salt
  per file. The file is written `0600` via an atomic rename.
- **Credentials are redacted by default** in `--dry-run` output, `auth status` and
  `config view`. `--show-token` reveals them only when you ask.
- **Cleartext HTTP is refused** for any non-loopback host, because an API token sent over
  plain HTTP is disclosed to every hop in between.
- **Secrets are read without echoing**, in raw terminal mode so a long pasted token cannot
  hit the canonical-mode buffer limit and hang.
- **Output is sanitized.** CSV cells are neutralized against spreadsheet formula injection
  (CWE-1236) and terminal escape sequences are stripped from API-supplied text, because
  issue summaries and page titles are attacker-controllable.
- **Writes are not blindly retried.** Only idempotent methods are re-sent, so a timeout can
  never create a duplicate issue or comment.

## Agent safety

`waba agent guard` generates host configuration that blocks irreversible operations and
gates writes. Its stated limits: it does not defeat variable indirection (`$X delete`) or
shell aliases and `eval`. Running an agent in MCP-only mode is the hard guarantee; the Bash
hook is defence in depth. See `docs/comparison.md` and `commands/agent_hosts.go`.
