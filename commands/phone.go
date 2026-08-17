package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("phone", "Manage business phone numbers", []string{"phones", "numbers"}, phoneSpecs)
}

var phoneColumns = []string{"id", "display_phone_number", "verified_name", "quality_rating", "code_verification_status"}

// resolvePhoneArg lets id-taking phone verbs default to the account's phone number, so
// `waba phone get` works without repeating the id every time.
func resolvePhoneArg(o *globalOptions, acct *config.Account, args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	return o.phoneNumberID(acct)
}

func phoneSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "list", Aliases: []string{"ls"}, Short: "List the WABA's phone numbers",
			Example: `  waba phone list
  waba phone list --fields id,display_phone_number,status,quality_rating`,
			Kind: kindRead, Args: cobra.NoArgs,
			Flags:   func(cmd *cobra.Command) { addListFlags(cmd) },
			Columns: phoneColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				waba, err := o.wabaID(acct)
				if err != nil {
					return nil, err
				}
				items, err := c.ListAll(cmd.Context(), waba+"/phone_numbers", nil, listParamsFrom(cmd))
				if err != nil {
					return nil, err
				}
				return rawList(items), nil
			},
		},
		{
			Use: "get [phone-number-id]", Short: "Show one phone number",
			Long:    "Defaults to the account's configured phone number when the id is omitted.",
			Example: `  waba phone get --fields id,display_phone_number,status,throughput,name_status`,
			Kind:    kindRead, Args: cobra.MaximumNArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("fields", "", "comma-separated field projection")
			},
			Columns: phoneColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				id, err := resolvePhoneArg(o, acct, args)
				if err != nil {
					return nil, err
				}
				q := urlValues()
				if f := mustString(cmd, "fields"); f != "" {
					q.Set("fields", f)
				}
				raw, err := c.Do(cmd.Context(), api.Request{Path: id, Query: q})
				if err != nil {
					return nil, err
				}
				return jsonMap(raw), nil
			},
		},
		{
			Use: "name-status [phone-number-id]", Short: "Check the display name review status",
			Kind: kindRead, Args: cobra.MaximumNArgs(1),
			Columns: []string{"id", "name_status"},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				id, err := resolvePhoneArg(o, acct, args)
				if err != nil {
					return nil, err
				}
				raw, err := c.Do(cmd.Context(), api.Request{Path: id, Query: urlValues("fields", "name_status")})
				if err != nil {
					return nil, err
				}
				return jsonMap(raw), nil
			},
		},
		{
			Use: "register [phone-number-id]", Short: "Register the number for Cloud API messaging",
			Long: "Registers the number so it can send messages, using the two-step verification PIN\n(set one first with `waba phone set-pin` if the number has none). Rate limited by Meta to\n10 attempts per number per 72 hours.",
			Example: `  waba phone register --pin 123456
  waba phone register --pin 123456 --region DE`,
			Kind: kindWrite, Args: cobra.MaximumNArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("pin", "", "6-digit two-step verification PIN")
				_ = cmd.MarkFlagRequired("pin")
				cmd.Flags().String("region", "", "data localization region (2-letter ISO country code)")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				id, err := resolvePhoneArg(o, acct, args)
				if err != nil {
					return nil, err
				}
				body := map[string]any{"messaging_product": "whatsapp", "pin": mustString(cmd, "pin")}
				if r := mustString(cmd, "region"); r != "" {
					body["data_localization_region"] = r
				}
				var out api.SuccessResult
				if err := c.PostJSON(cmd.Context(), id+"/register", body, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "registered %s", id)
				return nil, nil
			},
		},
		{
			Use: "deregister [phone-number-id]", Short: "Deregister the number from Cloud API",
			Long: "Disables Cloud API messaging for the number (reversible by registering again).",
			Kind: kindDestructive, Args: cobra.MaximumNArgs(1),
			Confirm: "Deregister phone number %s from the Cloud API?",
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				id, err := resolvePhoneArg(o, acct, args)
				if err != nil {
					return nil, err
				}
				var out api.SuccessResult
				if err := c.PostJSON(cmd.Context(), id+"/deregister", map[string]string{}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "deregistered %s", id)
				return nil, nil
			},
		},
		{
			Use: "request-code [phone-number-id]", Short: "Request an ownership verification code",
			Example: `  waba phone request-code --method SMS --language en_US`,
			Kind:    kindWrite, Args: cobra.MaximumNArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("method", "SMS", "delivery method: SMS or VOICE")
				cmd.Flags().String("language", "en_US", "language for the code message")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				id, err := resolvePhoneArg(o, acct, args)
				if err != nil {
					return nil, err
				}
				method := strings.ToUpper(mustString(cmd, "method"))
				if method != "SMS" && method != "VOICE" {
					return nil, fmt.Errorf("--method must be SMS or VOICE")
				}
				var out api.SuccessResult
				if err := c.PostJSON(cmd.Context(), id+"/request_code",
					map[string]string{"code_method": method, "language": mustString(cmd, "language")}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "verification code requested via %s", method)
				return nil, nil
			},
		},
		{
			Use: "verify-code <code> [phone-number-id]", Short: "Submit the received verification code",
			Kind: kindWrite, Args: cobra.RangeArgs(1, 2),
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				id, err := resolvePhoneArg(o, acct, args[1:])
				if err != nil {
					return nil, err
				}
				var out api.SuccessResult
				if err := c.PostJSON(cmd.Context(), id+"/verify_code", map[string]string{"code": args[0]}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "phone number verified")
				return nil, nil
			},
		},
		{
			Use: "set-pin <pin> [phone-number-id]", Short: "Set or change the two-step verification PIN",
			Long: "Sets the 6-digit two-step verification PIN. There is no API to disable two-step\nverification, and a forgotten PIN can only be reset from WhatsApp Manager.",
			Kind: kindWrite, Args: cobra.RangeArgs(1, 2),
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				if len(args[0]) != 6 {
					return nil, fmt.Errorf("the PIN must be exactly 6 digits")
				}
				id, err := resolvePhoneArg(o, acct, args[1:])
				if err != nil {
					return nil, err
				}
				var out api.SuccessResult
				if err := c.PostJSON(cmd.Context(), id, map[string]string{"pin": args[0]}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "two-step PIN updated")
				return nil, nil
			},
		},
		{
			Use: "settings [phone-number-id]", Short: "Show the number's settings (incl. calling and SIP)",
			Example: `  waba phone settings
  waba phone settings --sip-credentials`,
			Kind: kindRead, Args: cobra.MaximumNArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("sip-credentials", false, "include the SIP password in the response")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				id, err := resolvePhoneArg(o, acct, args)
				if err != nil {
					return nil, err
				}
				q := urlValues()
				if v, _ := cmd.Flags().GetBool("sip-credentials"); v {
					q.Set("include_sip_credentials", "true")
				}
				raw, err := c.Do(cmd.Context(), api.Request{Path: id + "/settings", Query: q})
				if err != nil {
					return nil, err
				}
				return jsonMap(raw), nil
			},
		},
		{
			Use: "update-settings [phone-number-id]", Short: "Update the number's settings (calling, SIP, identity checks)",
			Long: "Posts a settings document verbatim — the calling/SIP/voicemail shape is deeply nested\nand Meta evolves it, so it is passed as JSON rather than flags. Note: enabling SIP turns\noff the /calls endpoints and calling webhooks for the number.",
			Example: `  waba phone update-settings -d '{"calling":{"status":"ENABLED"}}'
  waba phone update-settings -d @calling-settings.json`,
			Kind: kindWrite, Args: cobra.MaximumNArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().StringP("data", "d", "", "settings document as JSON, @file, or @- for stdin")
				_ = cmd.MarkFlagRequired("data")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				id, err := resolvePhoneArg(o, acct, args)
				if err != nil {
					return nil, err
				}
				body, err := readJSONBody(mustString(cmd, "data"))
				if err != nil {
					return nil, err
				}
				var out api.SuccessResult
				if err := c.PostJSON(cmd.Context(), id+"/settings", body, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "settings updated (changes can take up to 7 days to reach users)")
				return nil, nil
			},
		},
	}
}
