package commands

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// `agent guard` turns this CLI's own command tree into safety configuration for whichever
// agent host is driving it: irreversible operations are blocked outright, ordinary writes
// require approval, reads run freely.
//
// It is derived from the live tree rather than a hand-maintained list, so it stays correct
// when commands are added — the failure mode of a static rule file is that it silently stops
// covering the thing it was written to cover.

// classification of a command for guard purposes.
const (
	classRead    = "read"    // safe, no approval needed
	classWrite   = "write"   // mutates remote state, needs approval
	classDestroy = "destroy" // irreversible, blocked outright
	classLocal   = "local"   // touches only this machine's config; never sent to Meta
)

// localGroups are the top-level commands that never mutate anything in Atlassian.
//
// This list is what `TestEveryAPICommandIsAnnotated` checks against: any command outside it
// that carries no annotation fails the build, so an unannotated new command can never fall
// through the classifier as harmless.
var localGroups = []string{
	"auth", "config", "alias", "completion", "version", "doctor", "init", "help", "mcp", "agent", "update",
}

// alwaysBlocked are verbs whose effect cannot be undone through this CLI. `auth logout` is
// deliberately absent: it destroys only a local credential, which is recoverable by logging
// in again. `block remove` (an unblock — reversible) is swept up by "remove"; denying a
// reversible write is accepted collateral, allowing an irreversible one is not.
var alwaysBlocked = []string{"delete", "rm", "remove", "deregister", "delete-by-id", "bulk-delete", "remove-participants", "unsubscribe", "deprecate"}

// guardCommand is one classified leaf of the tree.
type guardCommand struct {
	Path    string   // space-separated, without the binary name: "issues delete"
	Aliases []string // every alias spelling of the same leaf
	Class   string
}

// classifyCommands walks the tree and classifies every runnable leaf.
func classifyCommands(root *cobra.Command) []guardCommand {
	var out []guardCommand
	var walk func(cmd *cobra.Command, path []string)

	walk = func(cmd *cobra.Command, path []string) {
		for _, child := range cmd.Commands() {
			if child.Hidden || child.Name() == "help" {
				continue
			}
			childPath := append(append([]string{}, path...), child.Name())

			if child.Runnable() {
				out = append(out, guardCommand{
					Path:    strings.Join(childPath, " "),
					Aliases: aliasPaths(childPath, child, root),
					Class:   classify(child, childPath),
				})
			}
			walk(child, childPath)
		}
	}
	walk(root, nil)

	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// classify decides a single command's class.
//
// The order matters. A full-path override is checked before anything else because verb names
// collide across groups — `templates delete` destroys while `media delete` also destroys but
// `block remove` merely unblocks, and a name-based rule alone would get one of them wrong. An unannotated command outside the local groups
// falls through to `write`, never to `read`: guessing safe is the failure that matters.
func classify(cmd *cobra.Command, path []string) string {
	if len(path) == 0 {
		return classWrite
	}
	group := path[0]
	leaf := path[len(path)-1]

	// Commands that only touch local state, whatever their verb.
	if slices.Contains(localGroups, group) {
		return classLocal
	}

	// `api` is the raw escape hatch: it can issue any method against any path.
	if group == "api" {
		return classDestroy
	}

	if slices.Contains(alwaysBlocked, leaf) {
		return classDestroy
	}

	switch AnnotationKind(cmd) {
	case kindRead:
		return classRead
	case kindDestructive:
		return classDestroy
	case kindWrite:
		return classWrite
	}
	// No annotation and not a known-local group: treat as a write rather than assuming safe.
	return classWrite
}

// aliasPaths expands the group × verb alias cross-product for one leaf.
//
// Rules and hooks that list only canonical paths are trivially bypassed: `waba tpl delete`
// reaches the same code as `waba templates delete`. Every spelling has to be enumerated.
func aliasPaths(path []string, cmd *cobra.Command, root *cobra.Command) []string {
	// Collect the alias set for each segment, walking the real tree to find each ancestor.
	segments := make([][]string, len(path))
	node := root
	for i, name := range path {
		var found *cobra.Command
		for _, c := range node.Commands() {
			if c.Name() == name {
				found = c
				break
			}
		}
		spellings := []string{name}
		if found != nil {
			spellings = append(spellings, found.Aliases...)
			node = found
		}
		segments[i] = spellings
	}
	_ = cmd

	// Cartesian product of the per-segment spellings.
	combos := []string{""}
	for _, spellings := range segments {
		next := make([]string, 0, len(combos)*len(spellings))
		for _, prefix := range combos {
			for _, s := range spellings {
				if prefix == "" {
					next = append(next, s)
				} else {
					next = append(next, prefix+" "+s)
				}
			}
		}
		combos = next
	}
	sort.Strings(combos)
	return combos
}

// rawAPICommand is the one destroy-class command that must NOT be blocked as a whole path.
//
// `api` can issue any method, so it classifies as destructive — but blocking the bare path
// would also block `api GET`, which is a read. It is enforced instead by a method-position
// rule (see buildPreToolUseHook and apiBlockedMethods), so a GET whose PATH merely contains
// "delete" keeps working while a real DELETE does not.
const rawAPICommand = "api"

// apiBlockedMethods are the HTTP methods the raw escape hatch may not use.
var apiBlockedMethods = []string{"DELETE", "PUT", "POST", "PATCH"}

// blockedPaths returns every alias spelling of every destroy-class command, plus the local
// commands that can weaken the guard itself.
func blockedPaths(commands []guardCommand) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range commands {
		if c.Class != classDestroy {
			continue
		}
		// Handled by the method-position rule instead, so that `api GET` stays allowed.
		if c.Path == rawAPICommand {
			continue
		}
		for _, p := range c.Aliases {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	// `alias set` can point a harmless-looking name at a blocked command, which would defeat
	// every path-based rule below it.
	for _, p := range []string{"alias set", "alias remove", "alias rm"} {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// approvalPaths returns every alias spelling of every write-class command.
func approvalPaths(commands []guardCommand) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range commands {
		if c.Class != classWrite {
			continue
		}
		for _, p := range c.Aliases {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	sort.Strings(out)
	return out
}

func init() {
	registerMeta(func(root *cobra.Command, o *globalOptions) {
		root.AddCommand(newAgentCmd(o))
	})
}

func newAgentCmd(o *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Generate agent-host safety configuration from this CLI's own command tree",
	}
	cmd.AddCommand(newAgentGuardCmd(o), newAgentClassifyCmd(o))
	return cmd
}

func newAgentGuardCmd(o *globalOptions) *cobra.Command {
	var (
		host      string
		write     bool
		allWrites bool
		dir       string
	)

	cmd := &cobra.Command{
		Use:   "guard",
		Short: "Emit safety rules that block irreversible WhatsApp operations",
		Long: strings.TrimSpace(`
Generate safety configuration for an agent host driving this CLI.

Every runnable command is classified from the live tree: reads run freely, writes require
approval, and irreversible operations (delete/deregister/deprecate variants and the raw
'api' escape hatch, which can issue any method) are blocked outright.

The generated hook is the enforcement layer. Permission rules alone are literal prefixes, so
'./bin/waba templates delete', 'env X=1 waba templates delete' and quote-splitting all
walk straight past them; the hook matches the command position anywhere in the line.

Known limits, stated rather than papered over: variable indirection ($X delete) and shell
aliases or eval are not defeated. Running the agent in MCP-only mode is the hard guarantee;
the Bash hook is defence in depth.`),
		Example: strings.TrimSpace(`
  waba agent guard --host claude-code
  waba agent guard --host claude-code --write
  waba agent guard --host codex
  waba agent guard --host opencode --all-writes`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			commands := classifyCommands(cmd.Root())
			blocked := blockedPaths(commands)
			approvals := approvalPaths(commands)
			if allWrites {
				// Treat every write as irreversible too — the strict posture for unattended runs.
				blocked = append(blocked, approvals...)
				sort.Strings(blocked)
				approvals = nil
			}

			files, err := renderHostConfig(host, guardInput{
				Binary:     "waba",
				ToolPrefix: "waba",
				Blocked:    blocked,
				Approvals:  approvals,
			})
			if err != nil {
				return err
			}

			if !write {
				for _, f := range files {
					fmt.Fprintf(cmd.OutOrStdout(), "# %s\n%s\n", f.Path, f.Content)
				}
				o.note(cmd.ErrOrStderr(), "nothing written — re-run with --write to install these files")
				return nil
			}
			return writeGuardFiles(cmd, o, dir, files)
		},
	}

	cmd.Flags().StringVar(&host, "host", "claude-code", "agent host: claude-code, codex or opencode")
	cmd.Flags().BoolVar(&write, "write", false, "write the configuration files instead of printing them")
	cmd.Flags().BoolVar(&allWrites, "all-writes", false, "block every write, not only the irreversible ones")
	cmd.Flags().StringVar(&dir, "dir", "", "directory to write into (default: the host's own location)")
	annotate(cmd, kindRead)
	cmd.Annotations["wabaLocal"] = "true"
	return cmd
}

func newAgentClassifyCmd(o *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "classify",
		Short: "Show how each command is classified (read, write, destroy, local)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			commands := classifyCommands(cmd.Root())
			rows := make([]map[string]any, 0, len(commands))
			for _, c := range commands {
				rows = append(rows, map[string]any{
					"command": c.Path,
					"class":   c.Class,
					"aliases": strings.Join(c.Aliases, ", "),
				})
			}
			return o.renderList(cmd, rows, []string{"command", "class", "aliases"}, "command")
		},
	}
	annotate(cmd, kindRead)
	cmd.Annotations["wabaLocal"] = "true"
	return cmd
}
