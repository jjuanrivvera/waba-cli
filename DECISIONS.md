# DECISIONS.md — pinned assumptions for waba-cli

One line per decision: question → decision → why. The build loop reads this back instead of
re-deciding (cliwright GOAL.md §11).

1. **Binary name** → `waba`, repo `waba-cli` → fleet naming rule: repo says what it's for,
   binary says what it does; "waba" (WhatsApp Business Account) is the domain noun, no PATH
   collision found.
2. **Resource pattern** → service-layer over a generic Graph client (Pattern B) → documented
   trigger: Graph API edges are not CRUD-on-a-resource (register, publish, `?fields=`
   expansion params, payload-discriminated actions on one path); a `Resource[T]` CRUD core
   would fit almost none of the surface.
3. **Enumeration source** → no official OpenAPI/Postman JSON is downloadable without a
   Postman account, and no community machine spec exists → scraped the docs' full method
   index (Cloud API + Business Management API + Flows + Calling reference pages,
   developers.facebook.com, 2026-08-17) into `specs/enumeration.json`; the manifest is
   generated from it. Counting rule: one operation per method+path+distinctly-documented
   action; each per-type message page (text, image, …) counts separately because the
   reference nav documents them separately.
4. **Graph API version** → default `v25.0` (the version Meta's current doc examples use),
   configurable per account (`graph_version`) and overridable with `WABA_GRAPH_VERSION` →
   Meta versions the whole Graph API; pinning per account keeps behaviour stable.
5. **Profile noun** → `--account` (a profile bundles WABA id + phone number id + app id +
   token) with `--profile` kept as a hidden alias → GOAL.md §3 naming rule.
6. **Auth** → single method: static bearer access token (System User token in practice).
   No OAuth2/PKCE flow: Meta's embedded-signup OAuth targets platform apps, not a personal
   CLI; a pasted long-lived System User token is the documented integration path.
7. **Groups join_requests (3 ops)** → enumerated but NOT wrapped → the only param-level
   source is a third-party enumeration (Meta's reference page is JS-rendered and
   unscrapeable); wrapping would mean fabricating body shapes. Reachable via `waba api`.
   102/105 = 97% coverage, above the 90% gate — no waiver needed.
8. **`send address` message** → excluded from enumeration and CLI → not in the messages
   reference type enum; market-limited (India/Indonesia) interactive variant documented only
   in a guide. Reachable via `waba send interactive` with a hand-built payload.
9. **Welcome messages / ice-breaker `enable_welcome_message`** → NOT implemented → removed
   from the API; `conversational_automation` today carries only `commands` + `prompts`.
10. **Flexible JSON types** → keep the fleet's ID/Int/Bool/Number/StringOrSlice decoders →
    Graph API emits ids as strings but counts/percentages inconsistently across edges, and
    webhook-adjacent payloads mix string/number; the decoders absorb it.
11. **Rate limiting** → fixed-RPS floor (10 rps) + halve-on-429 with gradual restore, plus
    parsing Meta's `X-Business-Use-Case-Usage` percentage header to slow down near 100% →
    Cloud API publishes no numeric remaining-quota header; percentages are the only signal.
12. **Retry** → idempotent methods only (GET/HEAD/PUT/DELETE/OPTIONS) → a retried POST
    /messages would double-send a WhatsApp message; visibly failing beats silent duplicates.
13. **Media download host** → `media url` returns a lookaside URL on a different host;
    `media download` follows it with the same bearer token, restricted to hosts under
    facebook.com/fbcdn.net/fbsbx.com → sending the token to an arbitrary URL from a crafted
    response would be credential exfiltration.
14. **Resumable upload auth quirk** → `/upload:{session}` calls send `Authorization: OAuth
    <token>` (not `Bearer`) and a `file_offset` header → matches Meta's documented form;
    handled via a per-request auth-scheme override, not a second authenticator.
15. **Pagination** → Graph cursor pagination only (`paging.cursors.after` / `paging.next`);
    `--all` follows `after` cursors with the request's own path+query → following the
    absolute `next` URL verbatim would bypass base-URL/version handling.
16. **conversation_analytics granularity** → `HALF_HOUR|DAILY|MONTHLY` while messaging
    analytics uses `HALF_HOUR|DAY|MONTH` → per docs; the CLI validates each against its own
    enum rather than sharing one.
17. **distribution_scope** → `local-build` (build + local commits only; no remote repo, no
    push, no release) → the playbook default; publishing is the user's explicit call.
18. **Marketing Messages API** → `marketing send` wraps POST /{phone-number-id}/marketing_messages;
    the WABA-level toggles ride on `account get/update` → they are plain WABA node
    fields/params, not separate endpoints.
19. **Flow metrics deprecation** → `flows metrics` ships despite Meta's announced 2026-04-30
    discontinuation → it is still documented and live; the command help notes the sunset.
20. **Analytics via `?fields=` expansions** → messaging/conversation/pricing/call analytics
    are field expansions on GET /{waba-id}, not edges → the CLI builds the dot-notation
    field expression from flags; template/template-group/group analytics use their real
    edges.
