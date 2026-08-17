package commands

import (
	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("account", "Inspect and update the WhatsApp Business Account (WABA)", []string{"waba"}, accountSpecs)
	registerGroup("apps", "Webhook subscriptions (subscribed apps)", nil, appsSpecs)
}

const defaultWABAFields = "id,name,currency,timezone_id,message_template_namespace,account_review_status,business_verification_status,country,ownership_type"

func accountSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "get", Short: "Show the WABA node",
			Example: `  waba account get
  waba account get --fields id,name,health_status
  waba account get --fields disable_marketing_messages_on_cloud_api`,
			Kind: kindRead, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("fields", defaultWABAFields, "comma-separated field projection")
			},
			Columns: []string{"id", "name", "currency", "timezone_id", "country"},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				waba, err := o.wabaID(acct)
				if err != nil {
					return nil, err
				}
				raw, err := c.Do(cmd.Context(), api.Request{Path: waba, Query: urlValues("fields", mustString(cmd, "fields"))})
				if err != nil {
					return nil, err
				}
				return jsonMap(raw), nil
			},
		},
		{
			Use: "update", Short: "Update WABA-level settings",
			Long:    "Posts WABA node parameters verbatim — used for Marketing Messages toggles and other\naccount-level flags Meta adds over time.",
			Example: `  waba account update -d '{"disable_marketing_messages_on_cloud_api":true}'`,
			Kind:    kindWrite, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().StringP("data", "d", "", "parameters as JSON, @file, or @- for stdin")
				_ = cmd.MarkFlagRequired("data")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				waba, err := o.wabaID(acct)
				if err != nil {
					return nil, err
				}
				body, err := readJSONBody(mustString(cmd, "data"))
				if err != nil {
					return nil, err
				}
				var out api.SuccessResult
				if err := c.PostJSON(cmd.Context(), waba, body, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "WABA settings updated")
				return nil, nil
			},
		},
		{
			Use: "enable-insights", Short: "Enable template analytics on the WABA",
			Long: "Prerequisite for `waba analytics templates`. Meta documents this as NOT reversible\nvia the API once enabled.",
			Kind: kindWrite, Args: cobra.NoArgs,
			Confirm: "Enable template analytics? Meta documents this as irreversible via the API.%.0s",
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				waba, err := o.wabaID(acct)
				if err != nil {
					return nil, err
				}
				var out api.SuccessResult
				if err := c.DoInto(cmd.Context(), api.Request{Method: "POST", Path: waba,
					Query: urlValues("is_enabled_for_insights", "true")}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "template analytics enabled")
				return nil, nil
			},
		},
	}
}

var appColumns = []string{"id", "name", "link", "override_callback_uri"}

func appsSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "list", Aliases: []string{"ls"}, Short: "List apps subscribed to the WABA's webhooks",
			Kind: kindRead, Args: cobra.NoArgs,
			Columns: appColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				waba, err := o.wabaID(acct)
				if err != nil {
					return nil, err
				}
				var page struct {
					Data []struct {
						Data     api.Ref `json:"whatsapp_business_api_data"`
						Override string  `json:"override_callback_uri"`
					} `json:"data"`
				}
				if err := c.GetJSON(cmd.Context(), waba+"/subscribed_apps", nil, &page); err != nil {
					return nil, err
				}
				rows := make([]map[string]any, len(page.Data))
				for i, d := range page.Data {
					rows[i] = map[string]any{
						"id": d.Data.ID.String(), "name": d.Data.Name, "link": d.Data.Link,
						"override_callback_uri": d.Override,
					}
				}
				return rows, nil
			},
		},
		{
			Use: "subscribe", Short: "Subscribe the token's app to the WABA's webhooks",
			Long: "Subscribes the app the access token belongs to. An alternate callback URL (with its\nverify token) can override the app-level webhook for just this WABA.",
			Example: `  waba apps subscribe
  waba apps subscribe --callback-url https://bot.example.com/webhook --verify-token s3cret`,
			Kind: kindWrite, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("callback-url", "", "override callback URL for this WABA")
				cmd.Flags().String("verify-token", "", "verify token for the override callback")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				waba, err := o.wabaID(acct)
				if err != nil {
					return nil, err
				}
				var body any
				if cb := mustString(cmd, "callback-url"); cb != "" {
					body = map[string]string{"override_callback_uri": cb, "verify_token": mustString(cmd, "verify-token")}
				}
				var out api.SuccessResult
				if err := c.PostJSON(cmd.Context(), waba+"/subscribed_apps", body, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "app subscribed to WABA webhooks")
				return nil, nil
			},
		},
		{
			Use: "unsubscribe", Short: "Unsubscribe the token's app from the WABA's webhooks",
			Long: "Webhook delivery to the app stops immediately.",
			Kind: kindDestructive, Args: cobra.NoArgs,
			Confirm: "Unsubscribe the app from this WABA's webhooks?%.0s",
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				waba, err := o.wabaID(acct)
				if err != nil {
					return nil, err
				}
				var out api.SuccessResult
				if err := c.DoInto(cmd.Context(), api.Request{Method: "DELETE", Path: waba + "/subscribed_apps"}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "app unsubscribed")
				return nil, nil
			},
		},
	}
}
