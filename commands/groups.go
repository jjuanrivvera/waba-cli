package commands

import (
	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("groups", "WhatsApp groups (requires an Official Business Account)", []string{"group"}, groupSpecs)
}

var groupColumns = []string{"id", "subject", "description"}

func groupSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "create <subject>", Short: "Create a group",
			Long:    "Groups hold at most 8 participants and 10,000 groups per business number. Users join\nvia the invite link — there is no API to add participants directly.",
			Example: `  waba groups create "Clientes VIP" --description "Ofertas exclusivas"`,
			Kind:    kindWrite, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("description", "", "group description")
			},
			Columns: groupColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				body := map[string]any{"subject": args[0]}
				if d := mustString(cmd, "description"); d != "" {
					body["description"] = d
				}
				raw, err := c.Do(cmd.Context(), api.Request{Method: "POST", Path: phone + "/groups", Body: body})
				if err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "group created")
				return jsonMap(raw), nil
			},
		},
		{
			Use: "list", Aliases: []string{"ls"}, Short: "List the number's active groups",
			Kind: kindRead, Args: cobra.NoArgs,
			Flags:   func(cmd *cobra.Command) { addListFlags(cmd) },
			Columns: groupColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				items, err := c.ListAll(cmd.Context(), phone+"/groups", nil, listParamsFrom(cmd))
				if err != nil {
					return nil, err
				}
				return rawList(items), nil
			},
		},
		{
			Use: "get <group-id>", Short: "Show one group",
			Kind: kindRead, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("fields", "", "comma-separated field projection")
			},
			Columns: groupColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				q := urlValues()
				if f := mustString(cmd, "fields"); f != "" {
					q.Set("fields", f)
				}
				raw, err := c.Do(cmd.Context(), api.Request{Path: args[0], Query: q})
				if err != nil {
					return nil, err
				}
				return jsonMap(raw), nil
			},
		},
		{
			Use: "update <group-id>", Short: "Update a group's subject or description",
			Kind: kindWrite, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("subject", "", "new group subject")
				cmd.Flags().String("description", "", "new group description")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				body := map[string]any{}
				if v := mustString(cmd, "subject"); v != "" {
					body["subject"] = v
				}
				if v := mustString(cmd, "description"); v != "" {
					body["description"] = v
				}
				if len(body) == 0 {
					return nil, errNothingToUpdate
				}
				var out api.SuccessResult
				if err := c.PostJSON(cmd.Context(), args[0], body, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "group %s updated", args[0])
				return nil, nil
			},
		},
		{
			Use: "delete <group-id>", Short: "Delete a group",
			Kind: kindDestructive, Args: cobra.ExactArgs(1),
			Confirm: "Delete group %s?",
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				var out api.SuccessResult
				if err := c.DoInto(cmd.Context(), api.Request{Method: "DELETE", Path: args[0]}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "deleted group %s", args[0])
				return nil, nil
			},
		},
		{
			Use: "remove-participants <group-id> <number>...", Short: "Remove participants from a group",
			Kind: kindDestructive, Args: cobra.MinimumNArgs(2),
			Confirm: "Remove the listed participants from group %s?",
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				users := make([]map[string]string, len(args)-1)
				for i, n := range args[1:] {
					users[i] = map[string]string{"user": n}
				}
				raw, err := c.Do(cmd.Context(), api.Request{Method: "DELETE", Path: args[0] + "/participants",
					Body: map[string]any{"participants": users}})
				if err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "removed %d participant(s)", len(args)-1)
				return jsonMap(raw), nil
			},
		},
		{
			Use: "invite-link <group-id>", Short: "Get the group's invite link",
			Kind: kindRead, Args: cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				raw, err := c.Do(cmd.Context(), api.Request{Path: args[0] + "/invite_link"})
				if err != nil {
					return nil, err
				}
				return jsonMap(raw), nil
			},
		},
		{
			Use: "reset-invite-link <group-id>", Short: "Revoke the invite link and issue a new one",
			Kind: kindWrite, Args: cobra.ExactArgs(1),
			Confirm: "Reset the invite link for group %s? The current link stops working.",
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				raw, err := c.Do(cmd.Context(), api.Request{Method: "POST", Path: args[0] + "/invite_link"})
				if err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "invite link reset")
				return jsonMap(raw), nil
			},
		},
		{
			Use: "send <group-id> <text>", Short: "Send a text message to a group",
			Long:    "Groups accept text, media and text/media templates — interactive, authentication and\ncommerce messages are not supported. For media or templates, use `waba send … --to\n<group-id>` with recipient_type group via `waba api`.",
			Example: `  waba groups send 120363043211234567@g.us "La oferta termina hoy"`,
			Kind:    kindWrite, Args: cobra.ExactArgs(2),
			Columns: sendColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				payload := map[string]any{
					"messaging_product": "whatsapp",
					"recipient_type":    "group",
					"to":                args[0],
					"type":              "text",
					"text":              map[string]any{"body": args[1]},
				}
				res, err := c.SendMessage(cmd.Context(), phone, payload)
				if err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "sent to group %s", args[0])
				return flatSend(res), nil
			},
		},
		{
			Use: "pin <group-id> <wamid>", Short: "Pin a message in a group (admin only, max 3)",
			Kind: kindWrite, Args: cobra.ExactArgs(2),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().Int("days", 7, "how long the pin lasts (1–30 days)")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				days, _ := cmd.Flags().GetInt("days")
				payload := map[string]any{
					"messaging_product": "whatsapp",
					"recipient_type":    "group",
					"to":                args[0],
					"type":              "pin",
					"pin": map[string]any{
						"type":            "pin",
						"message_id":      args[1],
						"expiration_days": days,
					},
				}
				res, err := c.SendMessage(cmd.Context(), phone, payload)
				if err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "pinned %s for %d day(s)", args[1], days)
				return flatSend(res), nil
			},
		},
		{
			Use: "unpin <group-id> <wamid>", Short: "Unpin a message in a group",
			Kind: kindWrite, Args: cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				payload := map[string]any{
					"messaging_product": "whatsapp",
					"recipient_type":    "group",
					"to":                args[0],
					"type":              "unpin",
					"pin": map[string]any{
						"type":       "unpin",
						"message_id": args[1],
					},
				}
				res, err := c.SendMessage(cmd.Context(), phone, payload)
				if err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "unpinned %s", args[1])
				return flatSend(res), nil
			},
		},
	}
}
