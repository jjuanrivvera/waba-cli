#!/usr/bin/env bash
# gen-manifest.sh — derive api-manifest.json from the enumerated spec (GOAL.md §0/§11).
# specs/enumeration.json is the enumeration SOURCE; the manifest is generated, never
# hand-edited, so the surface stays reproducible run over run.
set -euo pipefail
cd "$(dirname "$0")/.."

jq '{
  api: "WhatsApp Cloud API",
  binary: "waba",
  base_url: "https://graph.facebook.com",
  graph_version: .graph_version,
  profile_flag: "account",
  profile_noun: "account",
  api_method_total: (.operations | length),
  api_method_source: ("developers.facebook.com reference method index (Cloud API + Business Management + Flows + Calling), scraped " + .scraped + " — specs/enumeration.json"),
  resources: (
    .operations
    | map(select(.wrapped))
    | group_by(.resource)
    | map({
        name: .[0].resource,
        verbs: (map(.verb)),
        kinds: (map({key: .verb, value: .kind}) | from_entries)
      })
    | sort_by(.name)
  ),
  unwrapped: (.operations | map(select(.wrapped | not)) | map(.resource + " " + .verb))
}' specs/enumeration.json > api-manifest.json
echo "✓ api-manifest.json: $(jq '.api_method_total' api-manifest.json) enumerated, $(jq '[.resources[].verbs | length] | add' api-manifest.json) wrapped"
