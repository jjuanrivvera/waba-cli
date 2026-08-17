package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/auth"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerMeta(func(root *cobra.Command, o *globalOptions) { root.AddCommand(newAuthCmd(o)) })
}

func newAuthCmd(o *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Log in, log out, and inspect authentication",
		Long: strings.TrimSpace(`
Manage the Graph API access token for an account. The token is stored in the OS keyring
(or the encrypted-file fallback on headless machines — set WABA_KEYRING_PASSWORD).

In practice the right credential is a System User token generated in Meta Business
Manager with the whatsapp_business_messaging and whatsapp_business_management
permissions; App Dashboard temporary tokens work but expire within 24 hours.`),
	}
	cmd.AddCommand(newAuthLoginCmd(o), newAuthLogoutCmd(o), newAuthStatusCmd(o))
	return cmd
}

func newAuthLoginCmd(o *globalOptions) *cobra.Command {
	var token string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store an access token and verify it against the API",
		Example: strings.TrimSpace(`
  # Interactive (the token prompt does not echo)
  waba auth login

  # Scripted
  waba auth login --token "$MY_TOKEN"`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			acct := resolveOrCreateAccount(cfg, o.account)

			if token == "" {
				token, err = promptSecret(cmd, "Access token (input hidden): ")
				if err != nil {
					return err
				}
			}
			if strings.TrimSpace(token) == "" {
				return fmt.Errorf("no token provided")
			}

			// Verify before storing: a mistyped token caught now is one confusing 190 later.
			ident, err := verifyToken(cmd, o, acct, token)
			if err != nil {
				return fmt.Errorf("token verification failed: %w", err)
			}

			if err := auth.NewStore().Set(acct.Name, auth.Credential{Token: token}); err != nil {
				return err
			}
			if _, ok := cfg.Accounts[acct.Name]; !ok {
				if err := cfg.Put(acct); err != nil {
					return err
				}
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			o.note(cmd.ErrOrStderr(), "logged in to account %q as %s", acct.Name, ident)
			return nil
		},
	}
	cmd.Flags().StringVar(&token, "token", "", "access token (omit to be prompted without echo)")
	cmd.Annotations = map[string]string{"wabaLocal": "true"}
	return cmd
}

func newAuthLogoutCmd(o *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove the stored token for the active account",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			acct, err := cfg.Resolve(o.account)
			if err != nil {
				return err
			}
			if err := auth.NewStore().Delete(acct.Name); err != nil {
				if err == auth.ErrNotFound {
					o.note(cmd.ErrOrStderr(), "no token was stored for account %q", acct.Name)
					return nil
				}
				return err
			}
			o.note(cmd.ErrOrStderr(), "logged out of account %q", acct.Name)
			return nil
		},
	}
	cmd.Annotations = map[string]string{"wabaLocal": "true"}
	return cmd
}

func newAuthStatusCmd(o *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"whoami"},
		Short:   "Show the active account, token backend and identity",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			acct, err := cfg.Resolve(o.account)
			if err != nil {
				return err
			}
			store := auth.NewStore()
			status := map[string]any{
				"account":       acct.Name,
				"base_url":      acct.BaseURL,
				"graph_version": acct.GraphVersion,
				"waba_id":       acct.WABAID,
				"phone_id":      acct.PhoneNumberID,
				"backend":       store.Backend(),
			}

			token, err := auth.ResolveToken(store, acct.Name)
			if err != nil {
				status["token"] = "missing"
				status["valid"] = false
			} else {
				status["token"] = "stored"
				if ident, err := verifyToken(cmd, o, acct, token); err != nil {
					status["valid"] = false
					status["error"] = err.Error()
				} else {
					status["valid"] = true
					status["identity"] = ident
				}
			}
			return o.render(cmd, status, []string{"account", "identity", "valid", "backend", "waba_id", "phone_id"})
		},
	}
	cmd.Annotations = map[string]string{"wabaLocal": "true"}
	return cmd
}

// verifyToken calls GET /me with an explicit token and returns a printable identity.
func verifyToken(cmd *cobra.Command, o *globalOptions, acct *config.Account, token string) (string, error) {
	client := api.NewClient(acct.BaseURL, acct.GraphVersion,
		api.WithAuthenticator(&auth.Bearer{Token: token}),
		api.WithRateLimit(0),
		api.WithTimeout(o.timeout),
	)
	var me struct {
		ID   api.ID `json:"id"`
		Name string `json:"name"`
	}
	if err := client.GetJSON(cmd.Context(), "me", urlValues("fields", "id,name"), &me); err != nil {
		return "", err
	}
	if me.Name != "" {
		return fmt.Sprintf("%s (id %s)", me.Name, me.ID), nil
	}
	return "id " + me.ID.String(), nil
}

// resolveOrCreateAccount picks the account login should attach to: the named/active one, or
// a fresh "default" so first-time `auth login` works without running init first.
func resolveOrCreateAccount(cfg *config.Config, explicit string) *config.Account {
	if acct, err := cfg.Resolve(explicit); err == nil && acct.Name != "env" {
		return acct
	}
	name := explicit
	if name == "" {
		name = "default"
	}
	return &config.Account{Name: name, BaseURL: config.DefaultBaseURL, GraphVersion: config.DefaultGraphVersion}
}
