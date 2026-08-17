package commands

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerMeta(func(root *cobra.Command, o *globalOptions) { root.AddCommand(newConfigCmd(o)) })
}

func newConfigCmd(o *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and edit the configuration",
		Long: strings.TrimSpace(`
Configuration lives in a YAML file (see 'waba config path') and holds only non-secret
settings: accounts with their WABA / phone number / app ids, the Graph version, and output
defaults. Access tokens are never written here — they live in the OS keyring.`),
	}
	cmd.AddCommand(
		newConfigPathCmd(),
		newConfigViewCmd(o),
		newConfigSetCmd(o),
		newConfigUseCmd(o),
		newConfigListAccountsCmd(o),
		newConfigRemoveCmd(o),
	)
	return cmd
}

func newConfigPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Print the config file location",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := config.Path()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), p)
			return nil
		},
	}
	cmd.Annotations = map[string]string{"wabaLocal": "true"}
	return cmd
}

func newConfigViewCmd(o *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view",
		Short: "Show the resolved configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			view := map[string]any{
				"path":            cfg.FilePath(),
				"current_account": cfg.CurrentAccount,
				"accounts":        cfg.Accounts,
				"output":          cfg.Output,
				"rate_limit":      cfg.RateLimit,
			}
			return o.render(cmd, view, nil)
		},
	}
	cmd.Annotations = map[string]string{"wabaLocal": "true"}
	return cmd
}

// settableFields maps `config set` keys to account fields. Kept explicit so a typo is an
// error, not a silently-ignored key.
var settableFields = []string{"waba_id", "phone_number_id", "app_id", "business_id", "graph_version", "base_url", "output", "rate_limit"}

func newConfigSetCmd(o *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set an account field or a global default",
		Long: "Keys: " + strings.Join(settableFields, ", ") + `.
Account fields apply to the active account (or --` + ProfileFlag + `); output and rate_limit are global.`,
		Example: strings.TrimSpace(`
  waba config set phone_number_id 106540352242922
  waba config set graph_version v25.0
  waba config set output json`),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			switch key {
			case "output":
				cfg.Output = value
			case "rate_limit":
				f, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return fmt.Errorf("rate_limit must be a number: %w", err)
				}
				cfg.RateLimit = f
			case "waba_id", "phone_number_id", "app_id", "business_id", "graph_version", "base_url":
				acct := storedAccountForEdit(cfg, o.account)
				switch key {
				case "waba_id":
					acct.WABAID = value
				case "phone_number_id":
					acct.PhoneNumberID = value
				case "app_id":
					acct.AppID = value
				case "business_id":
					acct.BusinessID = value
				case "graph_version":
					if err := config.ValidateGraphVersion(value); err != nil {
						return err
					}
					acct.GraphVersion = value
				case "base_url":
					if err := config.ValidateBaseURL(value); err != nil {
						return err
					}
					acct.BaseURL = value
				}
				if err := cfg.Put(acct); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unknown key %q (known: %s)", key, strings.Join(settableFields, ", "))
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			o.note(cmd.ErrOrStderr(), "set %s", key)
			return nil
		},
	}
	cmd.Annotations = map[string]string{"wabaLocal": "true"}
	return cmd
}

// storedAccountForEdit returns the RAW stored account to mutate — never the env-overlaid
// clone Resolve returns. Persisting a resolved clone would write WABA_* environment values
// into the config file, silently overwriting whatever the user had stored.
func storedAccountForEdit(cfg *config.Config, explicit string) *config.Account {
	name := config.FirstNonEmpty(explicit, os.Getenv(config.EnvPrefix+"ACCOUNT"), cfg.CurrentAccount)
	if name == "" && len(cfg.Accounts) == 1 {
		for n := range cfg.Accounts {
			name = n
		}
	}
	if name == "" || name == "env" {
		name = "default"
	}
	if a, ok := cfg.Accounts[name]; ok && a != nil {
		return a
	}
	return &config.Account{Name: name}
}

func newConfigUseCmd(o *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use <account>",
		Short: "Switch the active account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Accounts[args[0]]; !ok {
				return fmt.Errorf("unknown account %q (known: %s)", args[0], strings.Join(cfg.AccountNames(), ", "))
			}
			cfg.CurrentAccount = args[0]
			if err := cfg.Save(); err != nil {
				return err
			}
			o.note(cmd.ErrOrStderr(), "switched to account %q", args[0])
			return nil
		},
	}
	cmd.Annotations = map[string]string{"wabaLocal": "true"}
	return cmd
}

func newConfigListAccountsCmd(o *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list-accounts",
		Aliases: []string{"accounts"},
		Short:   "List configured accounts",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			rows := make([]map[string]any, 0, len(cfg.Accounts))
			for _, name := range cfg.AccountNames() {
				a := cfg.Accounts[name]
				rows = append(rows, map[string]any{
					"name":     name,
					"waba_id":  a.WABAID,
					"phone_id": a.PhoneNumberID,
					"version":  a.GraphVersion,
					"current":  name == cfg.CurrentAccount,
				})
			}
			sort.Slice(rows, func(i, j int) bool { return rows[i]["name"].(string) < rows[j]["name"].(string) })
			return o.renderList(cmd, rows, []string{"name", "waba_id", "phone_id", "version", "current"}, "name")
		},
	}
	cmd.Annotations = map[string]string{"wabaLocal": "true"}
	return cmd
}

func newConfigRemoveCmd(o *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <account>",
		Short: "Remove an account from the config (its token stays in the keyring until `auth logout`)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if !cfg.Remove(args[0]) {
				return fmt.Errorf("unknown account %q", args[0])
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			o.note(cmd.ErrOrStderr(), "removed account %q — run `waba auth logout --account %s` first next time to also drop its token", args[0], args[0])
			return nil
		},
	}
	cmd.Annotations = map[string]string{"wabaLocal": "true"}
	return cmd
}
