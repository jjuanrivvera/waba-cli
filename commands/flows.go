package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("flows", "Manage WhatsApp Flows", []string{"flow"}, flowSpecs)
}

var flowColumns = []string{"id", "name", "status", "categories", "validation_errors"}

func flowSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "list", Aliases: []string{"ls"}, Short: "List the WABA's flows",
			Kind: kindRead, Args: cobra.NoArgs,
			Flags:   func(cmd *cobra.Command) { addListFlags(cmd) },
			Columns: flowColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				waba, err := o.wabaID(acct)
				if err != nil {
					return nil, err
				}
				items, err := c.ListAll(cmd.Context(), waba+"/flows", nil, listParamsFrom(cmd))
				if err != nil {
					return nil, err
				}
				return rawList(items), nil
			},
		},
		{
			Use: "create <name>", Short: "Create a flow",
			Example: `  waba flows create "Agendar cita" --categories APPOINTMENT_BOOKING
  waba flows create "Encuesta" --categories SURVEY --json flow.json --publish`,
			Kind: kindWrite, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().StringSlice("categories", nil, "flow categories (e.g. SIGN_UP, APPOINTMENT_BOOKING, SURVEY, OTHER)")
				_ = cmd.MarkFlagRequired("categories")
				cmd.Flags().String("clone-from", "", "flow id to clone")
				cmd.Flags().String("endpoint-uri", "", "data-exchange endpoint URI")
				cmd.Flags().String("json", "", "path to a flow.json to attach on creation")
				cmd.Flags().Bool("publish", false, "publish immediately if the flow validates")
			},
			Columns: []string{"id", "success", "validation_errors"},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				waba, err := o.wabaID(acct)
				if err != nil {
					return nil, err
				}
				cats, _ := cmd.Flags().GetStringSlice("categories")
				body := map[string]any{"name": args[0], "categories": cats}
				if v := mustString(cmd, "clone-from"); v != "" {
					body["clone_flow_id"] = v
				}
				if v := mustString(cmd, "endpoint-uri"); v != "" {
					body["endpoint_uri"] = v
				}
				if path := mustString(cmd, "json"); path != "" {
					data, err := readFileForFlag(path)
					if err != nil {
						return nil, err
					}
					body["flow_json"] = string(data)
				}
				if v, _ := cmd.Flags().GetBool("publish"); v {
					body["publish"] = true
				}
				raw, err := c.Do(cmd.Context(), api.Request{Method: "POST", Path: waba + "/flows", Body: body})
				if err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "flow created")
				return jsonMap(raw), nil
			},
		},
		{
			Use: "get <flow-id>", Short: "Show one flow",
			Example: `  waba flows get 123456 --fields id,name,status,json_version,endpoint_uri`,
			Kind:    kindRead, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("fields", "", "comma-separated field projection")
			},
			Columns: flowColumns,
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
			Use: "preview <flow-id>", Short: "Get an embeddable preview URL (valid 30 days)",
			Kind: kindRead, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("invalidate", false, "force a fresh preview URL")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				inv, _ := cmd.Flags().GetBool("invalidate")
				expr := fmt.Sprintf("preview.invalidate(%t)", inv)
				raw, err := c.Do(cmd.Context(), api.Request{Path: args[0], Query: urlValues("fields", expr)})
				if err != nil {
					return nil, err
				}
				return jsonMap(raw), nil
			},
		},
		{
			Use: "metrics <flow-id>", Short: "Endpoint metrics for a published flow",
			Long:    "Meta has announced this Metrics API will be discontinued on 2026-04-30; it still\nworks today for published flows with an endpoint and ≥250 requests in the window.",
			Example: `  waba flows metrics 123456 --metric ENDPOINT_REQUEST_COUNT --since 2026-08-01 --until 2026-08-17`,
			Kind:    kindRead, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("metric", "ENDPOINT_REQUEST_COUNT", "ENDPOINT_REQUEST_COUNT|ENDPOINT_REQUEST_ERROR|ENDPOINT_REQUEST_ERROR_RATE|ENDPOINT_REQUEST_LATENCY_SECONDS_CEIL|ENDPOINT_AVAILABILITY")
				cmd.Flags().String("granularity", "DAY", "DAY|HOUR|LIFETIME")
				cmd.Flags().String("since", "", "window start (YYYY-MM-DD)")
				cmd.Flags().String("until", "", "window end (YYYY-MM-DD)")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				expr := api.NewAnalyticsExpr("metric").
					Arg("name", mustString(cmd, "metric")).
					Arg("granularity", strings.ToUpper(mustString(cmd, "granularity"))).
					Arg("since", mustString(cmd, "since")).
					Arg("until", mustString(cmd, "until"))
				raw, err := c.Do(cmd.Context(), api.Request{Path: args[0], Query: expr.FieldsQuery()})
				if err != nil {
					return nil, err
				}
				return jsonMap(raw), nil
			},
		},
		{
			Use: "update <flow-id>", Short: "Update flow metadata (name, categories, endpoint)",
			Example: `  waba flows update 123456 --name "Agendar visita" --endpoint-uri https://bot.example.com/flow`,
			Kind:    kindWrite, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("name", "", "new flow name")
				cmd.Flags().StringSlice("categories", nil, "replacement category list")
				cmd.Flags().String("endpoint-uri", "", "data-exchange endpoint URI (requires Flow JSON ≥ 3.0)")
				cmd.Flags().String("application-id", "", "app id to attach")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				body := map[string]any{}
				if v := mustString(cmd, "name"); v != "" {
					body["name"] = v
				}
				if cats, _ := cmd.Flags().GetStringSlice("categories"); len(cats) > 0 {
					body["categories"] = cats
				}
				if v := mustString(cmd, "endpoint-uri"); v != "" {
					body["endpoint_uri"] = v
				}
				if v := mustString(cmd, "application-id"); v != "" {
					body["application_id"] = v
				}
				if len(body) == 0 {
					return nil, errNothingToUpdate
				}
				var out api.SuccessResult
				if err := c.PostJSON(cmd.Context(), args[0], body, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "flow %s updated", args[0])
				return nil, nil
			},
		},
		{
			Use: "upload-json <flow-id> <flow.json>", Short: "Upload the flow's JSON definition",
			Example: `  waba flows upload-json 123456 ./flow.json`,
			Kind:    kindWrite, Args: cobra.ExactArgs(2),
			Columns: []string{"success", "validation_errors"},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				data, err := readFileForFlag(args[1])
				if err != nil {
					return nil, err
				}
				body, contentType, err := api.MultipartBody(
					map[string]string{"name": "flow.json", "asset_type": "FLOW_JSON"},
					"file", "flow.json", data, "application/json")
				if err != nil {
					return nil, err
				}
				var out map[string]any
				if err := c.PostMultipart(cmd.Context(), args[0]+"/assets", body, contentType, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "flow.json uploaded (%d bytes)", len(data))
				return out, nil
			},
		},
		{
			Use: "assets <flow-id>", Short: "List the flow's assets",
			Kind: kindRead, Args: cobra.ExactArgs(1),
			Columns: []string{"name", "asset_type", "download_url"},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				items, err := c.ListAll(cmd.Context(), args[0]+"/assets", nil, api.ListParams{All: true})
				if err != nil {
					return nil, err
				}
				return rawList(items), nil
			},
		},
		{
			Use: "publish <flow-id>", Short: "Publish a draft flow",
			Long: "DRAFT → PUBLISHED. Every validation error must be resolved first; published flows\ncan only be retired via deprecate.",
			Kind: kindWrite, Args: cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				var out api.SuccessResult
				if err := c.PostJSON(cmd.Context(), args[0]+"/publish", map[string]string{}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "flow %s published", args[0])
				return nil, nil
			},
		},
		{
			Use: "deprecate <flow-id>", Short: "Deprecate a published flow",
			Kind: kindWrite, Args: cobra.ExactArgs(1),
			Confirm: "Deprecate flow %s? Deprecated flows cannot be sent or opened.",
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				var out api.SuccessResult
				if err := c.PostJSON(cmd.Context(), args[0]+"/deprecate", map[string]string{}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "flow %s deprecated", args[0])
				return nil, nil
			},
		},
		{
			Use: "delete <flow-id>", Short: "Delete a DRAFT flow",
			Kind: kindDestructive, Args: cobra.ExactArgs(1),
			Confirm: "Delete draft flow %s?",
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				var out api.SuccessResult
				if err := c.DoInto(cmd.Context(), api.Request{Method: "DELETE", Path: args[0]}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "deleted flow %s", args[0])
				return nil, nil
			},
		},
		{
			Use: "migrate --from <source-waba-id>", Short: "Copy flows from another WABA",
			Long: "Copies (never moves) flows between WABAs owned by the same Meta business. Name\ncollisions are skipped per flow; published state is preserved; new ids are issued.",
			Example: `  waba flows migrate --from 111222333
  waba flows migrate --from 111222333 --names lead_gen,booking`,
			Kind: kindWrite, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("from", "", "source WABA id")
				_ = cmd.MarkFlagRequired("from")
				cmd.Flags().StringSlice("names", nil, "only migrate these flows (default: all)")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				waba, err := o.wabaID(acct)
				if err != nil {
					return nil, err
				}
				q := urlValues("source_waba_id", mustString(cmd, "from"))
				if names, _ := cmd.Flags().GetStringSlice("names"); len(names) > 0 {
					q.Set("source_flow_names", strings.Join(names, ","))
				}
				raw, err := c.Do(cmd.Context(), api.Request{Method: "POST", Path: waba + "/migrate_flows", Query: q})
				if err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "migration requested")
				return jsonMap(raw), nil
			},
		},
	}
}
