package commands

import (
	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("marketing", "Marketing Messages API (requires MM API onboarding)", nil, marketingSpecs)
}

func marketingSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "send", Short: "Send a marketing template via the Marketing Messages API",
			Long: "Sends a MARKETING template through /marketing_messages, Meta's delivery-optimized\nchannel (formerly \"MM Lite\"). The WABA must be onboarded to the Marketing Messages API;\nWABA-level toggles ride on `waba account update`.",
			Example: `  waba marketing send --to 573001112233 --name promo_agosto --lang es_MX --param "Juan"
  waba marketing send --to 573001112233 --name promo --lang es --components @components.json --fallback`,
			Kind: kindWrite, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				sendFlags(cmd)
				cmd.Flags().String("name", "", "template name")
				_ = cmd.MarkFlagRequired("name")
				cmd.Flags().String("lang", "", "template language code")
				_ = cmd.MarkFlagRequired("lang")
				cmd.Flags().StringArray("param", nil, "positional body parameter (repeatable, in order)")
				cmd.Flags().String("components", "", "components array as JSON, @file, or @- (overrides --param)")
				cmd.Flags().Bool("fallback", false, "product_policy CLOUD_API_FALLBACK instead of STRICT")
			},
			Columns: sendColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				to := mustString(cmd, "to")
				tpl := map[string]any{
					"name":     mustString(cmd, "name"),
					"language": map[string]string{"code": mustString(cmd, "lang")},
				}
				if compJSON := mustString(cmd, "components"); compJSON != "" {
					raw, err := readJSONBody(compJSON)
					if err != nil {
						return nil, err
					}
					tpl["components"] = raw
				} else if params, _ := cmd.Flags().GetStringArray("param"); len(params) > 0 {
					ps := make([]map[string]string, len(params))
					for i, p := range params {
						ps[i] = map[string]string{"type": "text", "text": p}
					}
					tpl["components"] = []map[string]any{{"type": "body", "parameters": ps}}
				}
				payload := api.MessagePayload(to, "template", tpl)
				if fb, _ := cmd.Flags().GetBool("fallback"); fb {
					payload["product_policy"] = "CLOUD_API_FALLBACK"
				}
				var out api.SendResult
				if err := c.PostJSON(cmd.Context(), phone+"/marketing_messages", payload, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "marketing template sent to %s", to)
				return flatSend(&out), nil
			},
		},
	}
}
