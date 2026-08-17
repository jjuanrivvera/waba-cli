package commands

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
)

func init() {
	registerAPI(func(root *cobra.Command, o *globalOptions) { root.AddCommand(newAPICmd(o)) })
}

// newAPICmd is the raw escape hatch: any authenticated Graph call, for the operations the
// curated commands don't wrap (e.g. groups join_requests — see DECISIONS.md #7). This is the
// one documented exception to "every command goes through a typed service".
func newAPICmd(o *globalOptions) *cobra.Command {
	var (
		data    string
		queries []string
		headers []string
	)
	cmd := &cobra.Command{
		Use:   "api <METHOD> <PATH>",
		Short: "Raw authenticated Graph API request",
		Long: `Perform an arbitrary Graph API request with the account's token and Graph version.
PATH is version-relative ("me", "{waba-id}/flows"). Honors --dry-run and -o.`,
		Example: strings.TrimSpace(`
  waba api GET me -q fields=id,name
  waba api GET 102290129340398/phone_numbers
  waba api POST 106540352242922/messages -d '{"messaging_product":"whatsapp","to":"...","type":"text","text":{"body":"hi"}}'
  waba api DELETE 102290129340398/message_templates -q name=old_template`),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := strings.ToUpper(args[0])
			switch method {
			case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions:
			default:
				return fmt.Errorf("unsupported method %q", args[0])
			}

			body, err := readJSONBody(data)
			if err != nil {
				return err
			}
			q := url.Values{}
			for _, kv := range queries {
				k, v, ok := strings.Cut(kv, "=")
				if !ok {
					return fmt.Errorf("query %q is not key=value", kv)
				}
				q.Add(k, v)
			}
			h := http.Header{}
			for _, kv := range headers {
				k, v, ok := strings.Cut(kv, ":")
				if !ok {
					return fmt.Errorf("header %q is not key:value", kv)
				}
				h.Set(strings.TrimSpace(k), strings.TrimSpace(v))
			}

			client, _, err := o.clientFor(cmd)
			if err != nil {
				return err
			}
			req := api.Request{Method: method, Path: args[1], Query: q, Headers: h}
			if body != nil {
				req.Body = body
			}
			raw, err := client.Do(cmd.Context(), req)
			if err != nil {
				return err
			}
			if len(raw) == 0 {
				return nil
			}
			return o.render(cmd, jsonMap(raw), nil)
		},
	}
	cmd.Flags().StringVarP(&data, "data", "d", "", "JSON request body, @file, or @- for stdin")
	cmd.Flags().StringArrayVarP(&queries, "query", "q", nil, "query parameter key=value (repeatable)")
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "extra header key:value (repeatable)")
	// The method is free-form, so classify by the worst case; the agent guard additionally
	// inspects the method argument at the Bash level.
	annotate(cmd, kindDestructive)
	return cmd
}
