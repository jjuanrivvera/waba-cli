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
	registerMeta(func(root *cobra.Command, o *globalOptions) { root.AddCommand(newInitCmd(o)) })
}

func newInitCmd(o *globalOptions) *cobra.Command {
	var (
		name    string
		token   string
		wabaID  string
		phoneID string
		appID   string
		bizID   string
	)
	cmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"setup"},
		Short:   "First-run wizard: token, WABA id, default phone number",
		Long: strings.TrimSpace(`
Set up an account interactively (or fully via flags for scripts). The wizard verifies the
token, stores it in the OS keyring, discovers the WABA's phone numbers so you can pick the
default sender, and smoke-tests the result.`),
		Example: strings.TrimSpace(`
  waba init
  waba init --name prod --token "$TOKEN" --waba-id 102290... --phone-number-id 106540...`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if name == "" {
				name, err = promptLine(cmd, "Account name [default]: ")
				if err != nil {
					return err
				}
				if name == "" {
					name = "default"
				}
			}
			if err := config.ValidateAccountName(name); err != nil {
				return err
			}

			if token == "" {
				token, err = promptSecret(cmd, "Access token (input hidden): ")
				if err != nil {
					return err
				}
			}
			if strings.TrimSpace(token) == "" {
				return fmt.Errorf("an access token is required — generate a System User token in Meta Business Manager")
			}

			acct := config.NewAccount(name)
			acct.WABAID, acct.PhoneNumberID, acct.AppID, acct.BusinessID = wabaID, phoneID, appID, bizID

			ident, err := verifyToken(cmd, o, acct, token)
			if err != nil {
				return fmt.Errorf("token verification failed: %w", err)
			}
			o.note(cmd.ErrOrStderr(), "token ok — authenticated as %s", ident)

			if acct.WABAID == "" {
				acct.WABAID, err = promptLine(cmd, "WhatsApp Business Account id (WABA id, blank to skip): ")
				if err != nil {
					return err
				}
			}

			// With a WABA id we can list its numbers, which beats asking the user to paste an
			// id they would otherwise have to dig out of Business Manager.
			client := api.NewClient(acct.BaseURL, acct.GraphVersion,
				api.WithAuthenticator(&auth.Bearer{Token: token}), api.WithRateLimit(0), api.WithTimeout(o.timeout))
			if acct.WABAID != "" && acct.PhoneNumberID == "" {
				var page struct {
					Data []struct {
						ID           api.ID `json:"id"`
						Display      string `json:"display_phone_number"`
						VerifiedName string `json:"verified_name"`
					} `json:"data"`
				}
				if err := client.GetJSON(cmd.Context(), acct.WABAID+"/phone_numbers", nil, &page); err != nil {
					// The most common stumble here is pasting the PHONE NUMBER id at the
					// WABA prompt (both are opaque 15-16 digit numbers, shown side by side
					// in the App Dashboard). Probe for it and self-correct instead of
					// leaving a broken account behind.
					var probe struct {
						Display string `json:"display_phone_number"`
					}
					if perr := client.GetJSON(cmd.Context(), acct.WABAID,
						urlValues("fields", "display_phone_number"), &probe); perr == nil && probe.Display != "" {
						o.note(cmd.ErrOrStderr(), "%s is a phone number id (%s), not a WABA id — storing it as the default phone number. Find the WABA id in App Dashboard > WhatsApp > API Setup, then run `waba config set waba_id <id>`",
							acct.WABAID, probe.Display)
						acct.PhoneNumberID = acct.WABAID
						acct.WABAID = ""
					} else {
						o.note(cmd.ErrOrStderr(), "could not list phone numbers: %v", err)
					}
				} else {
					for _, p := range page.Data {
						fmt.Fprintf(cmd.ErrOrStderr(), "  %s  %s  (%s)\n", p.ID, p.Display, p.VerifiedName)
					}
					if len(page.Data) == 1 {
						acct.PhoneNumberID = page.Data[0].ID.String()
						o.note(cmd.ErrOrStderr(), "using the only phone number: %s", acct.PhoneNumberID)
					} else if len(page.Data) > 1 {
						acct.PhoneNumberID, err = promptLine(cmd, "Default phone number id: ")
						if err != nil {
							return err
						}
					}
				}
			}

			if err := auth.NewStore().Set(name, auth.Credential{Token: token}); err != nil {
				return err
			}
			if err := cfg.Put(acct); err != nil {
				return err
			}
			cfg.CurrentAccount = name
			if err := cfg.Save(); err != nil {
				return err
			}
			o.note(cmd.ErrOrStderr(), "account %q ready — try: waba phone list", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "account name (prompted if omitted)")
	cmd.Flags().StringVar(&token, "token", "", "access token (prompted without echo if omitted)")
	cmd.Flags().StringVar(&wabaID, "waba-id", "", "WhatsApp Business Account id")
	cmd.Flags().StringVar(&phoneID, "phone-number-id", "", "default business phone number id")
	cmd.Flags().StringVar(&appID, "app-id", "", "Meta app id (needed only for resumable uploads)")
	cmd.Flags().StringVar(&bizID, "business-id", "", "Meta business portfolio id (optional)")
	cmd.Annotations = map[string]string{"wabaLocal": "true"}
	return cmd
}
