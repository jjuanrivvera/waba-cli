# Output and filtering

## Formats

`-o table` (default, colored on a TTY only), `-o json`, `-o yaml`, `-o csv`, `-o id`
(one id per line — pipeable to xargs).

```sh
waba templates list -o json | jq '.[].name'
waba phone list -o id | xargs -I{} waba phone get {}
waba templates list -o csv > templates.csv     # cells are formula-injection sanitized
```

`--columns id,name,status` selects table/CSV columns; wide cells truncate with a stderr
hint to use `-o json`. `--quiet` suppresses notes; notes and warnings always go to stderr
so stdout stays pipe-clean. `NO_COLOR` and `--no-color` are honoured.

## Built-in jq

`--jq` runs a gojq expression on the result before rendering — no external jq needed:

```sh
waba templates list --jq '.[] | select(.status=="REJECTED") | .name'
waba analytics messaging --start 2026-08-01 --end 2026-08-17 --granularity DAY \
  --jq '.analytics.data_points'
```

## Pagination

Graph list edges paginate with opaque cursors. `--limit N` sizes a page, `--all` walks
every page, `--after <cursor>` resumes. Without `--all` only the first page returns.

## Dry runs

`--dry-run` on any command prints the exact `curl` equivalent — headers, payload, URL —
with the Authorization header redacted (`--show-token` reveals it) and performs no I/O.
Use it to show the user what would happen, or to hand off a reproducible request.

## Rate limits and retries

The client paces itself (`--rps`, default 10/s), halves on 429 and recovers gradually,
honours `Retry-After`, and slows down when Meta's `X-Business-Use-Case-Usage` header nears
100%. Only idempotent methods (GET/PUT/DELETE) are retried — a send is never re-posted.
`-v` traces requests, timings and usage headers to stderr.
