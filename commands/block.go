package commands

import (
	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("block", "Manage the blocklist", nil, blockSpecs)
}

// blockBody builds the shared block/unblock payload.
func blockBody(numbers []string) map[string]any {
	users := make([]map[string]string, len(numbers))
	for i, n := range numbers {
		users[i] = map[string]string{"user": n}
	}
	return map[string]any{"messaging_product": "whatsapp", "block_users": users}
}

func blockSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "add <number>...", Short: "Block users (max 1,000 per call)",
			Long:    "Only users who messaged the business in the last 24 hours can be blocked; the\nblocklist holds at most 64,000 entries.",
			Example: `  waba block add 573001112233 573004445566`,
			Kind:    kindWrite, Args: cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				raw, err := c.Do(cmd.Context(), api.Request{Method: "POST", Path: phone + "/block_users", Body: blockBody(args)})
				if err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "block request sent for %d user(s)", len(args))
				return jsonMap(raw), nil
			},
		},
		{
			Use: "remove <number>...", Short: "Unblock users",
			Kind: kindWrite, Args: cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				raw, err := c.Do(cmd.Context(), api.Request{Method: "DELETE", Path: phone + "/block_users", Body: blockBody(args)})
				if err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "unblock request sent for %d user(s)", len(args))
				return jsonMap(raw), nil
			},
		},
		{
			Use: "list", Aliases: []string{"ls"}, Short: "List blocked users",
			Kind: kindRead, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) { addListFlags(cmd) },
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				items, err := c.ListAll(cmd.Context(), phone+"/block_users", nil, listParamsFrom(cmd))
				if err != nil {
					return nil, err
				}
				return rawList(items), nil
			},
		},
	}
}
