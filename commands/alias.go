package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/shlex"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jjuanrivvera/waba-cli/internal/config"
)

// User-defined aliases, expanded BEFORE cobra parses the command line — cobra has no hook
// that runs early enough, and rewriting args afterwards would fight its own flag parsing.

func init() {
	registerMeta(func(root *cobra.Command, o *globalOptions) {
		root.AddCommand(newAliasCmd(o))
	})
}

// aliasFile is the on-disk map of alias name to expansion.
type aliasFile struct {
	Aliases map[string]string `yaml:"aliases"`
}

func aliasPath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "aliases.yaml"), nil
}

func loadAliases() (map[string]string, error) {
	path, err := aliasPath()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path) // #nosec G304 -- the CLI's own config location
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var f aliasFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if f.Aliases == nil {
		f.Aliases = map[string]string{}
	}
	return f.Aliases, nil
}

func saveAliases(m map[string]string) error {
	path, err := aliasPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	out, err := yaml.Marshal(aliasFile{Aliases: m})
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o600)
}

// ExpandAlias rewrites argv when its first element is a user alias.
//
// Called from main before cobra.Execute. Built-in commands always win, so an alias can never
// shadow (or silently redefine) a real command such as `delete`.
func ExpandAlias(args []string, builtins map[string]bool) []string {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return args
	}
	if builtins[args[0]] {
		return args
	}
	aliases, err := loadAliases()
	if err != nil {
		return args
	}
	expansion, ok := aliases[args[0]]
	if !ok {
		return args
	}
	parts, err := shlex.Split(expansion)
	if err != nil || len(parts) == 0 {
		return args
	}
	return append(parts, args[1:]...)
}

// BuiltinNames lists every top-level command and alias, so alias expansion knows what it
// must not shadow.
func BuiltinNames(root *cobra.Command) map[string]bool {
	out := map[string]bool{}
	for _, c := range root.Commands() {
		out[c.Name()] = true
		for _, a := range c.Aliases {
			out[a] = true
		}
	}
	return out
}

func newAliasCmd(o *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alias",
		Short: "Define shortcuts for longer commands",
		Example: strings.TrimSpace(`
  waba alias set approved "templates list --status APPROVED"
  waba approved
  waba alias list`),
	}
	cmd.AddCommand(newAliasSetCmd(o), newAliasListCmd(o), newAliasRemoveCmd(o))
	return cmd
}

func newAliasSetCmd(_ *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <name> <expansion>",
		Short: "Create or replace an alias",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, expansion := args[0], args[1]
			if strings.HasPrefix(name, "-") || strings.ContainsAny(name, " \t") {
				return fmt.Errorf("alias name %q must be a single word and cannot start with '-'", name)
			}
			if BuiltinNames(cmd.Root())[name] {
				return fmt.Errorf("%q is a built-in command and cannot be aliased", name)
			}
			aliases, err := loadAliases()
			if err != nil {
				return err
			}
			aliases[name] = expansion
			if err := saveAliases(aliases); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "alias %s = %s\n", name, expansion)
			return nil
		},
	}
	annotate(cmd, kindWrite)
	cmd.Annotations["wabaLocal"] = "true"
	return cmd
}

func newAliasListCmd(o *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List defined aliases",
		RunE: func(cmd *cobra.Command, _ []string) error {
			aliases, err := loadAliases()
			if err != nil {
				return err
			}
			names := make([]string, 0, len(aliases))
			for n := range aliases {
				names = append(names, n)
			}
			sort.Strings(names)

			rows := make([]map[string]string, 0, len(names))
			for _, n := range names {
				rows = append(rows, map[string]string{"alias": n, "expansion": aliases[n]})
			}
			return o.render(cmd, rows, []string{"alias", "expansion"})
		},
	}
	annotate(cmd, kindRead)
	cmd.Annotations["wabaLocal"] = "true"
	return cmd
}

func newAliasRemoveCmd(_ *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Short:   "Remove an alias",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			aliases, err := loadAliases()
			if err != nil {
				return err
			}
			if _, ok := aliases[args[0]]; !ok {
				return fmt.Errorf("no alias named %q", args[0])
			}
			delete(aliases, args[0])
			if err := saveAliases(aliases); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed alias %s\n", args[0])
			return nil
		},
	}
	annotate(cmd, kindDestructive)
	cmd.Annotations["wabaLocal"] = "true"
	return cmd
}
