package commands

import (
	"slices"

	"github.com/njayp/ophis"
	"github.com/spf13/cobra"
)

// The MCP server exposes this CLI's commands as annotated MCP tools, so an agent drives
// the WhatsApp Cloud API through the same client, keyring, account profiles, retry policy and --dry-run as a
// human at the terminal — rather than through a second, separately-maintained integration.

// mcpExcludedGroups are the top-level commands whose whole subtree stays off the MCP surface.
//
// Matching is on the top-level group name EXACTLY, never as a substring: a substring match on
// "update" would also drop every `<resource> update` tool and silently remove the write
// surface an agent is supposed to have.
var mcpExcludedGroups = []string{
	"agent",      // an agent must not be able to rewrite or disable its own safety rails
	"auth",       // credential capture belongs to the human
	"config",     // switching accounts or base URLs out from under the running server
	"init",       // same
	"alias",      // an alias could re-point a safe-looking name at a destructive command
	"completion", // shell plumbing, meaningless over MCP
	"mcp",        // no recursion
	"update",     // self-replacing the binary is not an agent's decision
	"doctor",     // local diagnostics that echo credential detail
	"version",
}

// mcpExcludedFlags never reach a tool schema.
//
// The server runs as whichever account was active at startup. Exposing the account selector
// (under both its real name and its hidden alias), the id overrides or --base-url would let
// an agent point the same tools at a different WABA or phone number; --show-token would let
// it read the credential back out of a dry run.
var mcpExcludedFlags = []string{
	"show-token",
	ProfileFlag, // "account"
	"profile",   // the hidden back-compat alias for --account
	"base-url",
	"phone-id",
	"waba-id",
	"app-id",
}

// mcpCommandSelector accepts a command unless its TOP-LEVEL group is excluded.
//
// Only the top-level name is compared, deliberately. Testing every node on the way up would
// also match a leaf that happens to share a name with an excluded group — `update` is both
// the binary's self-updater and the verb on several resources, so that version silently
// drops `profile update`, `flows update`, `commerce update` and the rest of the write
// surface.
// The subtree of an excluded group is still fully covered, because every command inside it
// resolves to the same top-level name.
func mcpCommandSelector(cmd *cobra.Command) bool {
	top := cmd
	for top.HasParent() && top.Parent().HasParent() {
		top = top.Parent()
	}
	if !top.HasParent() {
		return false // the root itself is not a tool
	}
	return !slices.Contains(mcpExcludedGroups, top.Name())
}

func init() {
	registerMeta(func(root *cobra.Command, _ *globalOptions) {
		root.AddCommand(ophis.Command(&ophis.Config{
			ToolNamePrefix: "waba",
			Selectors: []ophis.Selector{{
				CmdSelector:           mcpCommandSelector,
				InheritedFlagSelector: ophis.ExcludeFlags(mcpExcludedFlags...),
			}},
		}))
	})
}
