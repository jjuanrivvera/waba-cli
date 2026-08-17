package commands

import (
	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("profile", "Manage the WhatsApp Business profile", nil, profileSpecs)
}

const profileFields = "about,address,description,email,profile_picture_url,websites,vertical"

var profileColumns = []string{"about", "address", "description", "email", "websites", "vertical"}

func profileSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "get", Short: "Show the business profile",
			Example: `  waba profile get
  waba profile get --fields about,websites`,
			Kind: kindRead, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("fields", profileFields, "comma-separated field projection")
			},
			Columns: profileColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				var page struct {
					Data []map[string]any `json:"data"`
				}
				if err := c.GetJSON(cmd.Context(), phone+"/whatsapp_business_profile",
					urlValues("fields", mustString(cmd, "fields")), &page); err != nil {
					return nil, err
				}
				if len(page.Data) == 0 {
					return map[string]any{}, nil
				}
				return page.Data[0], nil
			},
		},
		{
			Use: "update", Short: "Update business profile fields",
			Long: "Updates only the fields you pass. The profile picture takes a handle from the\nresumable upload API (`waba uploads start` + `waba uploads upload`), not a URL.",
			Example: `  waba profile update --about "Reparación de neveras en Bogotá" --vertical OTHER
  waba profile update --website https://rivera-refrigeracion.com --email hola@example.com`,
			Kind: kindWrite, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("about", "", "the about text (max 139 chars)")
				cmd.Flags().String("address", "", "business address")
				cmd.Flags().String("description", "", "business description")
				cmd.Flags().String("email", "", "contact email")
				cmd.Flags().StringArray("website", nil, "website URL (repeatable, max 2)")
				cmd.Flags().String("vertical", "", "industry vertical (e.g. OTHER, RETAIL, PROF_SERVICES)")
				cmd.Flags().String("picture-handle", "", "profile picture handle from the resumable upload API")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				body := map[string]any{"messaging_product": "whatsapp"}
				for flag, key := range map[string]string{
					"about": "about", "address": "address", "description": "description",
					"email": "email", "vertical": "vertical", "picture-handle": "profile_picture_handle",
				} {
					if v := mustString(cmd, flag); v != "" {
						body[key] = v
					}
				}
				if sites, _ := cmd.Flags().GetStringArray("website"); len(sites) > 0 {
					body["websites"] = sites
				}
				var out api.SuccessResult
				if err := c.PostJSON(cmd.Context(), phone+"/whatsapp_business_profile", body, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "profile updated")
				return nil, nil
			},
		},
	}
}
