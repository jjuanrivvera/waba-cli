package commands

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("commerce", "Cart and catalog visibility settings", nil, commerceSpecs)
}

var commerceColumns = []string{"id", "is_cart_enabled", "is_catalog_visible"}

func commerceSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "get", Short: "Show commerce settings",
			Kind: kindRead, Args: cobra.NoArgs,
			Columns: commerceColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				var page struct {
					Data []map[string]any `json:"data"`
				}
				if err := c.GetJSON(cmd.Context(), phone+"/whatsapp_commerce_settings", nil, &page); err != nil {
					return nil, err
				}
				if len(page.Data) == 1 {
					return page.Data[0], nil
				}
				return page.Data, nil
			},
		},
		{
			Use: "update", Short: "Toggle the cart and catalog visibility",
			Example: `  waba commerce update --cart=false
  waba commerce update --catalog=true --cart=true`,
			Kind: kindWrite, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("cart", true, "enable the shopping cart")
				cmd.Flags().Bool("catalog", false, "show the catalog on the profile")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				// Only flags the user actually set are sent, so a cart toggle can't silently
				// reset catalog visibility to its default.
				q := urlValues()
				if cmd.Flags().Changed("cart") {
					v, _ := cmd.Flags().GetBool("cart")
					q.Set("is_cart_enabled", strconv.FormatBool(v))
				}
				if cmd.Flags().Changed("catalog") {
					v, _ := cmd.Flags().GetBool("catalog")
					q.Set("is_catalog_visible", strconv.FormatBool(v))
				}
				if len(q) == 0 {
					return nil, errNothingToUpdate
				}
				var out api.SuccessResult
				if err := c.DoInto(cmd.Context(), api.Request{Method: "POST", Path: phone + "/whatsapp_commerce_settings", Query: q}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "commerce settings updated")
				return nil, nil
			},
		},
	}
}
