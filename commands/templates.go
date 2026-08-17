package commands

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("templates", "Manage message templates", []string{"template", "tpl"}, templateSpecs)
}

var templateColumns = []string{"id", "name", "language", "category", "status", "quality_score"}

func templateSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "list", Aliases: []string{"ls"}, Short: "List the WABA's templates",
			Example: `  waba templates list --status APPROVED
  waba templates list --name order --all`,
			Kind: kindRead, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				addListFlags(cmd)
				cmd.Flags().String("status", "", "filter by status: APPROVED|PENDING|REJECTED|PAUSED|DISABLED")
				cmd.Flags().String("name", "", "filter by (partial) template name")
				cmd.Flags().String("language", "", "filter by language code")
				cmd.Flags().String("category", "", "filter by category: AUTHENTICATION|MARKETING|UTILITY")
			},
			Columns: templateColumns,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				waba, err := o.wabaID(acct)
				if err != nil {
					return nil, err
				}
				q := urlValues()
				for _, f := range []string{"status", "name", "language", "category"} {
					if v := mustString(cmd, f); v != "" {
						q.Set(f, v)
					}
				}
				p := listParamsFrom(cmd)
				if p.Fields == "" {
					p.Fields = "id,name,language,category,status,quality_score"
				}
				items, err := c.ListAll(cmd.Context(), waba+"/message_templates", q, p)
				if err != nil {
					return nil, err
				}
				return rawList(items), nil
			},
		},
		{
			Use: "get <template-id>", Short: "Show one template (components included)",
			Kind: kindRead, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().String("fields", "", "comma-separated field projection")
			},
			Columns: templateColumns,
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
			Use: "create", Short: "Create a template (goes to Meta review)",
			Long: "Creates a template from a JSON document — the components array (header, body,\nfooter, buttons) is too rich for flags. Meta reviews new templates before they can be\nsent; creation is limited to 100 templates per WABA per hour.",
			Example: `  waba templates create -d '{"name":"order_update","language":"es_MX","category":"UTILITY","components":[{"type":"BODY","text":"Hola {{1}}, tu pedido {{2}} va en camino."}]}'
  waba templates create -d @welcome.json`,
			Kind: kindWrite, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().StringP("data", "d", "", "template document as JSON, @file, or @- for stdin")
				_ = cmd.MarkFlagRequired("data")
			},
			Columns: []string{"id", "status", "category"},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				waba, err := o.wabaID(acct)
				if err != nil {
					return nil, err
				}
				body, err := readJSONBody(mustString(cmd, "data"))
				if err != nil {
					return nil, err
				}
				raw, err := c.Do(cmd.Context(), api.Request{Method: "POST", Path: waba + "/message_templates", Body: body})
				if err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "template submitted for review")
				return jsonMap(raw), nil
			},
		},
		{
			Use: "edit <template-id>", Short: "Edit a template (re-triggers review)",
			Long:    "Only APPROVED, REJECTED or PAUSED templates can be edited; approved templates allow\n10 edits per 30 days (1 per 24h).",
			Example: `  waba templates edit 1234567890 -d '{"components":[...]}'`,
			Kind:    kindWrite, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().StringP("data", "d", "", "fields to change as JSON, @file, or @- for stdin")
				_ = cmd.MarkFlagRequired("data")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				body, err := readJSONBody(mustString(cmd, "data"))
				if err != nil {
					return nil, err
				}
				var out api.SuccessResult
				if err := c.PostJSON(cmd.Context(), args[0], body, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "template %s updated", args[0])
				return nil, nil
			},
		},
		{
			Use: "delete <name>", Short: "Delete a template by name — ALL its languages",
			Long: "Deleting by name removes every language variant of the template. To delete a single\nlanguage, use `waba templates delete-by-id`.",
			Kind: kindDestructive, Args: cobra.ExactArgs(1),
			Confirm: "Delete template %q in ALL its languages?",
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				waba, err := o.wabaID(acct)
				if err != nil {
					return nil, err
				}
				var out api.SuccessResult
				if err := c.DoInto(cmd.Context(), api.Request{Method: "DELETE", Path: waba + "/message_templates",
					Query: urlValues("name", args[0])}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "deleted template %q (all languages)", args[0])
				return nil, nil
			},
		},
		{
			Use: "delete-by-id <template-id> <name>", Short: "Delete one language variant by id",
			Long: "The API requires BOTH the template id (hsm_id) and its name for a single-language\ndeletion.",
			Kind: kindDestructive, Args: cobra.ExactArgs(2),
			Confirm: "Delete template %s (single language)?",
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				waba, err := o.wabaID(acct)
				if err != nil {
					return nil, err
				}
				var out api.SuccessResult
				if err := c.DoInto(cmd.Context(), api.Request{Method: "DELETE", Path: waba + "/message_templates",
					Query: urlValues("hsm_id", args[0], "name", args[1])}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "deleted template %s", args[0])
				return nil, nil
			},
		},
		{
			Use: "bulk-delete <template-id>...", Short: "Delete up to 100 templates by id",
			Long: "All-or-nothing: one invalid id fails the whole request.",
			Kind: kindDestructive, Args: cobra.RangeArgs(1, 100),
			Confirm: "Delete %s and the other listed templates?",
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				waba, err := o.wabaID(acct)
				if err != nil {
					return nil, err
				}
				ids, err := json.Marshal(args)
				if err != nil {
					return nil, err
				}
				var out api.SuccessResult
				if err := c.DoInto(cmd.Context(), api.Request{Method: "DELETE", Path: waba + "/message_templates",
					Query: urlValues("hsm_ids", string(ids))}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "deleted %d template(s)", len(args))
				return nil, nil
			},
		},
		{
			Use: "compare <template-id> <other-template-id>", Short: "Compare two templates' performance",
			Long:    "Both templates must belong to the same WABA and have been sent ≥1,000 times in the\nwindow. The lookback is exactly 7, 30, 60 or 90 days ending now.",
			Example: `  waba templates compare 111 222 --days 30`,
			Kind:    kindRead, Args: cobra.ExactArgs(2),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().Int("days", 7, "lookback window: 7, 30, 60 or 90 days")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				days, _ := cmd.Flags().GetInt("days")
				switch days {
				case 7, 30, 60, 90:
				default:
					return nil, fmt.Errorf("--days must be 7, 30, 60 or 90")
				}
				end := nowUnix()
				start := end - int64(days)*24*3600
				raw, err := c.Do(cmd.Context(), api.Request{Path: args[0] + "/compare", Query: urlValues(
					"template_ids", args[1], "start", strconv.FormatInt(start, 10), "end", strconv.FormatInt(end, 10))})
				if err != nil {
					return nil, err
				}
				return jsonMap(raw), nil
			},
		},
		{
			Use: "click-tracking <template-id>", Short: "Toggle CTA URL click tracking for a template",
			Example: `  waba templates click-tracking 1234567890 --opt-out --category MARKETING`,
			Kind:    kindWrite, Args: cobra.ExactArgs(1),
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().Bool("opt-out", false, "opt the template OUT of click tracking (omit to opt back in)")
				cmd.Flags().String("category", "", "the template's category (required by the API for this action)")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				optOut, _ := cmd.Flags().GetBool("opt-out")
				q := urlValues("cta_url_link_tracking_opted_out", strconv.FormatBool(optOut))
				if cat := mustString(cmd, "category"); cat != "" {
					q.Set("category", cat)
				}
				var out api.SuccessResult
				if err := c.DoInto(cmd.Context(), api.Request{Method: "POST", Path: args[0], Query: q}, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "click tracking opted_out=%t for template %s", optOut, args[0])
				return nil, nil
			},
		},
	}
}
