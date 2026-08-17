package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

// The Graph API is edge-shaped rather than CRUD-shaped (DECISIONS.md #2), so instead of a
// CRUD generator there is one generic op builder: every resource verb declares an opSpec and
// gets the client wiring, dry-run behaviour, confirmation prompts, output rendering and MCP
// annotations from a single place. A resource file is just a group + a list of specs.

// annotation kinds, mapped to the MCP hint keys ophis understands.
//
// There is no "write" key in MCP: a write is expressed as openWorldHint set with
// readOnlyHint absent. Getting this wrong makes a host treat a mutating tool as safe.
const (
	kindRead        = "read"
	kindWrite       = "write"
	kindDestructive = "destructive"
)

// annotate stamps MCP tool hints on a command as it is built.
//
// Doing this in the builder — rather than in a later pass over the finished tree — is what
// keeps every generated subcommand correctly classified; a retrofit reliably misses some,
// and a missed destructive command is one an agent may run unattended.
func annotate(cmd *cobra.Command, kind string) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	switch kind {
	case kindRead:
		cmd.Annotations["readOnlyHint"] = "true"
		cmd.Annotations["openWorldHint"] = "true"
	case kindWrite:
		cmd.Annotations["openWorldHint"] = "true"
		cmd.Annotations["idempotentHint"] = "false"
	case kindDestructive:
		cmd.Annotations["destructiveHint"] = "true"
		cmd.Annotations["openWorldHint"] = "true"
	}
	cmd.Annotations["wabaKind"] = kind
}

// AnnotationKind reads back the classification the guard and MCP surface rely on.
func AnnotationKind(cmd *cobra.Command) string {
	if cmd.Annotations == nil {
		return ""
	}
	return cmd.Annotations["wabaKind"]
}

// opSpec declares one API operation as a subcommand.
type opSpec struct {
	Use     string
	Aliases []string
	Short   string
	Long    string
	Example string
	Kind    string // kindRead | kindWrite | kindDestructive
	Args    cobra.PositionalArgs

	// Flags adds op-specific flags before the command is finalized.
	Flags func(cmd *cobra.Command)

	// Confirm, when non-empty, prompts before running (destructive ops). %s is args[0].
	// --yes and --dry-run both skip the prompt.
	Confirm string

	// Columns are the preferred table columns for the rendered result.
	Columns []string

	// Run performs the call and returns the value to render (nil renders nothing; the op
	// prints its own note instead).
	Run func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error)
}

// newOpCmd builds a cobra command from an opSpec.
func newOpCmd(o *globalOptions, spec opSpec) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     spec.Use,
		Aliases: spec.Aliases,
		Short:   spec.Short,
		Long:    spec.Long,
		Example: spec.Example,
		Args:    spec.Args,
		RunE: func(cmd *cobra.Command, args []string) error {
			if spec.Confirm != "" && !yes && !o.dryRun {
				subject := ""
				if len(args) > 0 {
					subject = args[0]
				}
				ok, err := confirm(cmd, fmt.Sprintf(spec.Confirm, subject))
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("aborted")
				}
			}
			client, acct, err := o.clientFor(cmd)
			if err != nil {
				return err
			}
			v, err := spec.Run(cmd, o, client, acct, args)
			if err != nil {
				return err
			}
			if v == nil {
				return nil
			}
			return o.render(cmd, v, spec.Columns)
		},
	}
	if spec.Confirm != "" {
		cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation prompt")
	}
	if spec.Flags != nil {
		spec.Flags(cmd)
	}
	annotate(cmd, spec.Kind)
	return cmd
}

// registerGroup queues a resource group whose subcommands are opSpecs.
func registerGroup(use, short string, aliases []string, specs func(o *globalOptions) []opSpec) {
	registerAPI(func(root *cobra.Command, o *globalOptions) {
		group := &cobra.Command{Use: use, Aliases: aliases, Short: short}
		for _, s := range specs(o) {
			group.AddCommand(newOpCmd(o, s))
		}
		root.AddCommand(group)
	})
}

// renderList renders a collection with the resource's preferred columns and id field.
func (o *globalOptions) renderList(cmd *cobra.Command, items any, columns []string, idField string) error {
	if o.jq != "" {
		filtered, err := applyJQ(o.jq, items)
		if err != nil {
			return err
		}
		items = filtered
	}
	r := o.renderer(cmd, columns)
	if idField != "" {
		r.IDField = idField
	}
	return r.Render(items)
}

// addListFlags wires the standard pagination flags onto a list command; listParamsFrom
// reads them back at run time. Split into two stateless halves because an opSpec's Flags
// and Run are separate closures with no shared frame.
func addListFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("all", false, "fetch every page")
	cmd.Flags().Int("limit", 0, "items per page")
	cmd.Flags().String("after", "", "continue from a pagination cursor")
	cmd.Flags().String("before", "", "page backwards from a cursor")
	cmd.Flags().String("fields", "", "comma-separated field projection")
}

func listParamsFrom(cmd *cobra.Command) api.ListParams {
	all, _ := cmd.Flags().GetBool("all")
	limit, _ := cmd.Flags().GetInt("limit")
	after, _ := cmd.Flags().GetString("after")
	before, _ := cmd.Flags().GetString("before")
	fields, _ := cmd.Flags().GetString("fields")
	return api.ListParams{All: all, Limit: limit, After: after, Before: before, Fields: fields}
}

// rawList converts raw JSON items into renderable values.
func rawList(items []json.RawMessage) []any {
	rows := make([]any, 0, len(items))
	for _, raw := range items {
		var v any
		if err := json.Unmarshal(raw, &v); err == nil {
			rows = append(rows, v)
		}
	}
	return rows
}

// readJSONBody resolves a --data value: inline JSON, @file, or @- for stdin.
//
// The file form is deliberately unrestricted — the user names it directly on their own
// command line, which is not the confused-deputy case that path confinement guards against.
func readJSONBody(v string) (json.RawMessage, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	raw := []byte(v)
	switch {
	case v == "@-":
		b, err := readAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read body from stdin: %w", err)
		}
		raw = b
	case strings.HasPrefix(v, "@"):
		b, err := os.ReadFile(v[1:]) // #nosec G304 -- the path is supplied directly by the user
		if err != nil {
			return nil, fmt.Errorf("read body file: %w", err)
		}
		raw = b
	}
	if !json.Valid(raw) {
		return nil, fmt.Errorf("request body is not valid JSON")
	}
	return json.RawMessage(raw), nil
}

// readFileForFlag reads a file named directly on the command line by the user.
//
// No path confinement here on purpose: the user typed the path themselves, which is not the
// confused-deputy case that confinement protects against.
func readFileForFlag(path string) ([]byte, error) {
	b, err := os.ReadFile(path) // #nosec G304 -- path supplied directly by the user on the CLI
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return b, nil
}
