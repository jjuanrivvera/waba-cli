package commands

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("analytics", "Messaging, conversation, pricing, template and call analytics", []string{"insights"}, analyticsSpecs)
}

// windowFlags adds the --start/--end/--granularity trio every analytics edge takes.
func windowFlags(cmd *cobra.Command, defGranularity string, allowed []string) {
	cmd.Flags().String("start", "", "window start: Unix timestamp or YYYY-MM-DD")
	cmd.Flags().String("end", "", "window end: Unix timestamp or YYYY-MM-DD")
	_ = cmd.MarkFlagRequired("start")
	_ = cmd.MarkFlagRequired("end")
	cmd.Flags().String("granularity", defGranularity, "bucket size: "+strings.Join(allowed, "|"))
}

// window reads and validates the trio. The granularity enums genuinely differ per edge
// (DAY vs DAILY — see DECISIONS.md #16), so each caller passes its own allowed set.
func window(cmd *cobra.Command, allowed []string) (start, end int64, granularity string, err error) {
	start, err = parseTimeArg(mustString(cmd, "start"))
	if err != nil {
		return 0, 0, "", err
	}
	end, err = parseTimeArg(mustString(cmd, "end"))
	if err != nil {
		return 0, 0, "", err
	}
	granularity = strings.ToUpper(mustString(cmd, "granularity"))
	if !slices.Contains(allowed, granularity) {
		return 0, 0, "", fmt.Errorf("--granularity must be one of %s", strings.Join(allowed, "|"))
	}
	return start, end, granularity, nil
}

// fieldsExpansionRun runs a `GET /{waba-id}?fields=<expr>` analytics expansion.
func fieldsExpansionRun(field string, allowed []string, extra func(cmd *cobra.Command, e *api.AnalyticsExpr)) func(*cobra.Command, *globalOptions, *api.Client, *config.Account, []string) (any, error) {
	return func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
		waba, err := o.wabaID(acct)
		if err != nil {
			return nil, err
		}
		start, end, gran, err := window(cmd, allowed)
		if err != nil {
			return nil, err
		}
		expr := api.NewAnalyticsExpr(field).
			Arg("start", strconv.FormatInt(start, 10)).
			Arg("end", strconv.FormatInt(end, 10)).
			Arg("granularity", gran)
		if phones, _ := cmd.Flags().GetStringSlice("phone-numbers"); len(phones) > 0 {
			expr.List("phone_numbers", phones)
		}
		if extra != nil {
			extra(cmd, expr)
		}
		raw, err := c.Do(cmd.Context(), api.Request{Path: waba, Query: expr.FieldsQuery()})
		if err != nil {
			return nil, err
		}
		return jsonMap(raw), nil
	}
}

func analyticsSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "messaging", Short: "Sent and delivered message counts",
			Example: `  waba analytics messaging --start 2026-08-01 --end 2026-08-17 --granularity DAY`,
			Kind:    kindRead, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				windowFlags(cmd, "DAY", []string{"HALF_HOUR", "DAY", "MONTH"})
				cmd.Flags().StringSlice("phone-numbers", nil, "filter to these phone numbers")
				cmd.Flags().StringSlice("country-codes", nil, "filter to these ISO country codes")
			},
			Run: fieldsExpansionRun("analytics", []string{"HALF_HOUR", "DAY", "MONTH"},
				func(cmd *cobra.Command, e *api.AnalyticsExpr) {
					if cc, _ := cmd.Flags().GetStringSlice("country-codes"); len(cc) > 0 {
						e.List("country_codes", cc)
					}
				}),
		},
		{
			Use: "conversations", Short: "Conversation counts and costs",
			Example: `  waba analytics conversations --start 2026-08-01 --end 2026-08-17 --granularity DAILY \
    --dimensions CONVERSATION_CATEGORY,COUNTRY`,
			Kind: kindRead, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				windowFlags(cmd, "DAILY", []string{"HALF_HOUR", "DAILY", "MONTHLY"})
				cmd.Flags().StringSlice("phone-numbers", nil, "filter to these phone numbers")
				cmd.Flags().StringSlice("metric-types", nil, "COST and/or CONVERSATION")
				cmd.Flags().StringSlice("conversation-categories", nil, "AUTHENTICATION, MARKETING, SERVICE, UTILITY")
				cmd.Flags().StringSlice("conversation-types", nil, "FREE_ENTRY_POINT, FREE_TIER, REGULAR")
				cmd.Flags().StringSlice("conversation-directions", nil, "BUSINESS_INITIATED, USER_INITIATED")
				cmd.Flags().StringSlice("dimensions", nil, "breakdown dimensions (e.g. CONVERSATION_CATEGORY, COUNTRY, PHONE)")
			},
			Run: fieldsExpansionRun("conversation_analytics", []string{"HALF_HOUR", "DAILY", "MONTHLY"},
				func(cmd *cobra.Command, e *api.AnalyticsExpr) {
					for flag, name := range map[string]string{
						"metric-types": "metric_types", "conversation-categories": "conversation_categories",
						"conversation-types": "conversation_types", "conversation-directions": "conversation_directions",
						"dimensions": "dimensions",
					} {
						if vs, _ := cmd.Flags().GetStringSlice(flag); len(vs) > 0 {
							e.List(name, vs)
						}
					}
				}),
		},
		{
			Use: "pricing", Short: "Per-message pricing analytics",
			Example: `  waba analytics pricing --start 2026-08-01 --end 2026-08-17 --granularity DAILY --dimensions PRICING_CATEGORY`,
			Kind:    kindRead, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				windowFlags(cmd, "DAILY", []string{"HALF_HOUR", "DAILY", "MONTHLY"})
				cmd.Flags().StringSlice("phone-numbers", nil, "filter to these phone numbers")
				cmd.Flags().StringSlice("country-codes", nil, "filter to these ISO country codes")
				cmd.Flags().StringSlice("metric-types", nil, "COST and/or VOLUME")
				cmd.Flags().StringSlice("pricing-types", nil, "FREE_CUSTOMER_SERVICE, FREE_ENTRY_POINT, REGULAR")
				cmd.Flags().StringSlice("pricing-categories", nil, "AUTHENTICATION, MARKETING, SERVICE, UTILITY, …")
				cmd.Flags().StringSlice("dimensions", nil, "breakdown dimensions (COUNTRY, PHONE, PRICING_CATEGORY, …)")
			},
			Run: fieldsExpansionRun("pricing_analytics", []string{"HALF_HOUR", "DAILY", "MONTHLY"},
				func(cmd *cobra.Command, e *api.AnalyticsExpr) {
					for flag, name := range map[string]string{
						"country-codes": "country_codes", "metric-types": "metric_types",
						"pricing-types": "pricing_types", "pricing-categories": "pricing_categories",
						"dimensions": "dimensions",
					} {
						if vs, _ := cmd.Flags().GetStringSlice(flag); len(vs) > 0 {
							e.List(name, vs)
						}
					}
				}),
		},
		{
			Use: "calls", Short: "Calling API analytics",
			Example: `  waba analytics calls --start 2026-08-01 --end 2026-08-17 --granularity DAILY --metric-types COUNT,COST`,
			Kind:    kindRead, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				windowFlags(cmd, "DAILY", []string{"HALF_HOUR", "DAILY", "MONTHLY"})
				cmd.Flags().StringSlice("phone-numbers", nil, "filter to these phone numbers")
				cmd.Flags().StringSlice("country-codes", nil, "filter to these ISO country codes")
				cmd.Flags().StringSlice("directions", nil, "USER_INITIATED and/or BUSINESS_INITIATED")
				cmd.Flags().StringSlice("metric-types", nil, "COUNT, COST, AVERAGE_DURATION")
				cmd.Flags().StringSlice("dimensions", nil, "breakdown dimensions (phone, direction, country)")
			},
			Run: fieldsExpansionRun("call_analytics", []string{"HALF_HOUR", "DAILY", "MONTHLY"},
				func(cmd *cobra.Command, e *api.AnalyticsExpr) {
					for flag, name := range map[string]string{
						"country-codes": "country_codes", "directions": "directions",
						"metric-types": "metric_types", "dimensions": "dimensions",
					} {
						if vs, _ := cmd.Flags().GetStringSlice(flag); len(vs) > 0 {
							e.List(name, vs)
						}
					}
				}),
		},
		{
			Use: "templates", Short: "Per-template analytics (sent, delivered, read, clicked)",
			Long:    "Requires insights enabled on the WABA (`waba account enable-insights`).",
			Example: `  waba analytics templates --start 2026-08-01 --end 2026-08-17 --template-ids 111,222`,
			Kind:    kindRead, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				windowFlags(cmd, "DAILY", []string{"DAILY"})
				cmd.Flags().StringSlice("template-ids", nil, "template ids to report on")
				cmd.Flags().StringSlice("metric-types", nil, "SENT, DELIVERED, READ, CLICKED, COST, …")
				cmd.Flags().Bool("waba-timezone", false, "bucket days in the WABA's timezone instead of UTC")
			},
			Run: edgeAnalyticsRun("template_analytics", "template_ids", "template-ids"),
		},
		{
			Use: "template-groups", Short: "Template group analytics",
			Kind: kindRead, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				windowFlags(cmd, "DAILY", []string{"DAILY"})
				cmd.Flags().StringSlice("group-ids", nil, "template group ids to report on")
				cmd.Flags().StringSlice("metric-types", nil, "cost, clicked, delivered, read, sent")
				cmd.Flags().Bool("waba-timezone", false, "bucket days in the WABA's timezone instead of UTC")
			},
			Run: edgeAnalyticsRun("template_group_analytics", "template_group_ids", "group-ids"),
		},
		{
			Use: "groups", Short: "WhatsApp groups analytics",
			Kind: kindRead, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				windowFlags(cmd, "DAILY", []string{"DAILY"})
				cmd.Flags().StringSlice("group-ids", nil, "group ids to report on")
				cmd.Flags().StringSlice("metric-types", nil, "SENT, DELIVERED, READ, PARTICIPANTS_JOINED, PARTICIPANTS_LEFT")
			},
			Run: edgeAnalyticsRun("group_analytics", "group_ids", "group-ids"),
		},
	}
}

// edgeAnalyticsRun runs the analytics edges that are real GET endpoints (not fields=
// expansions): /template_analytics, /template_group_analytics, /group_analytics.
func edgeAnalyticsRun(edge, idsParam, idsFlag string) func(*cobra.Command, *globalOptions, *api.Client, *config.Account, []string) (any, error) {
	return func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
		waba, err := o.wabaID(acct)
		if err != nil {
			return nil, err
		}
		start, end, gran, err := window(cmd, []string{"DAILY"})
		if err != nil {
			return nil, err
		}
		q := urlValues(
			"start", strconv.FormatInt(start, 10),
			"end", strconv.FormatInt(end, 10),
			"granularity", gran,
		)
		if ids, _ := cmd.Flags().GetStringSlice(idsFlag); len(ids) > 0 {
			enc, err := json.Marshal(ids)
			if err != nil {
				return nil, err
			}
			q.Set(idsParam, string(enc))
		}
		if mt, _ := cmd.Flags().GetStringSlice("metric-types"); len(mt) > 0 {
			enc, err := json.Marshal(mt)
			if err != nil {
				return nil, err
			}
			q.Set("metric_types", string(enc))
		}
		if cmd.Flags().Lookup("waba-timezone") != nil {
			if v, _ := cmd.Flags().GetBool("waba-timezone"); v {
				q.Set("use_waba_timezone", "true")
			}
		}
		raw, err := c.Do(cmd.Context(), api.Request{Path: waba + "/" + edge, Query: q})
		if err != nil {
			return nil, err
		}
		return jsonMap(raw), nil
	}
}
