package commands

import (
	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("messages", "Mark messages read and show typing indicators", []string{"msg"}, messagesSpecs)
}

func messagesSpecs(o *globalOptions) []opSpec {
	// Both operations POST to /messages with a status payload; they differ only in the
	// typing_indicator block, so they share one runner.
	statusRun := func(typing bool) func(*cobra.Command, *globalOptions, *api.Client, *config.Account, []string) (any, error) {
		return func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
			phone, err := o.phoneNumberID(acct)
			if err != nil {
				return nil, err
			}
			payload := map[string]any{
				"messaging_product": "whatsapp",
				"status":            "read",
				"message_id":        args[0],
			}
			if typing {
				payload["typing_indicator"] = map[string]string{"type": "text"}
			}
			var out api.SuccessResult
			if err := c.PostJSON(cmd.Context(), phone+"/messages", payload, &out); err != nil {
				return nil, err
			}
			if typing {
				o.noteWrite(cmd.ErrOrStderr(), "marked read + typing shown (dismisses in ≤25s or on reply)")
			} else {
				o.noteWrite(cmd.ErrOrStderr(), "marked read")
			}
			return nil, nil
		}
	}

	return []opSpec{
		{
			Use: "read <wamid>", Short: "Mark an inbound message as read",
			Long:    "Marks the message — and earlier messages in the same conversation — as read (blue\nticks). The wamid comes from the messages webhook and is valid for 30 days.",
			Example: `  waba messages read wamid.HBgLNTczMDA...`,
			Kind:    kindWrite, Args: cobra.ExactArgs(1),
			Run: statusRun(false),
		},
		{
			Use: "typing <wamid>", Short: "Mark read and show a typing indicator",
			Long:    "Combines the read receipt with a typing indicator, which stays visible for up to 25\nseconds or until the next message is sent.",
			Example: `  waba messages typing wamid.HBgLNTczMDA...`,
			Kind:    kindWrite, Args: cobra.ExactArgs(1),
			Run: statusRun(true),
		},
	}
}
