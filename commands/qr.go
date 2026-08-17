package commands

import (
	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("qr", "Manage QR codes and short links (wa.me deep links)", nil, qrSpecs)
}

var qrColumns = []string{"code", "prefilled_message", "deep_link_url", "qr_image_url"}

func qrSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "create <message>", Short: "Create a QR code with a prefilled message",
			Example: `  waba qr create "Hola! Quiero más información" --image SVG`,
			Kind:    kindWrite, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("image", "", "also return a rendered QR image: SVG or PNG")
			},
			Columns: qrColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				body := map[string]string{"prefilled_message": args[0]}
				if img := mustString(cmd, "image"); img != "" {
					body["generate_qr_image"] = img
				}
				raw, err := c.Do(cmd.Context(), api.Request{Method: "POST", Path: phone + "/message_qrdls", Body: body})
				if err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "QR code created")
				return jsonMap(raw), nil
			},
		},
		{
			Use: "list", Aliases: []string{"ls"}, Short: "List the number's QR codes",
			Kind: kindRead, Args: cobra.NoArgs,
			Columns: qrColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				var page struct {
					Data []map[string]any `json:"data"`
				}
				if err := c.GetJSON(cmd.Context(), phone+"/message_qrdls", nil, &page); err != nil {
					return nil, err
				}
				return page.Data, nil
			},
		},
		{
			Use: "get <code-id>", Short: "Show one QR code",
			Kind: kindRead, Args: cobra.ExactArgs(1),
			Columns: qrColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				var page struct {
					Data []map[string]any `json:"data"`
				}
				if err := c.GetJSON(cmd.Context(), phone+"/message_qrdls/"+args[0], nil, &page); err != nil {
					return nil, err
				}
				if len(page.Data) == 1 {
					return page.Data[0], nil
				}
				return page.Data, nil
			},
		},
		{
			Use: "update <code-id> <message>", Short: "Change a QR code's prefilled message",
			Long: "Updates via POST to the collection with a `code` body param — the API has no\nper-code update path.",
			Kind: kindWrite, Args: cobra.ExactArgs(2),
			Columns: qrColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				raw, err := c.Do(cmd.Context(), api.Request{Method: "POST", Path: phone + "/message_qrdls",
					Body: map[string]string{"code": args[0], "prefilled_message": args[1]}})
				if err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "QR code %s updated", args[0])
				return jsonMap(raw), nil
			},
		},
		{
			Use: "delete <code-id>", Short: "Retire a QR code permanently",
			Kind: kindDestructive, Args: cobra.ExactArgs(1),
			Confirm: "Permanently retire QR code %s?",
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				var out api.SuccessResult
				if err := c.DoInto(cmd.Context(), api.Request{Method: "DELETE", Path: phone + "/message_qrdls/" + args[0]}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "deleted QR code %s", args[0])
				return nil, nil
			},
		},
	}
}
