package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/auth"
	"github.com/jjuanrivvera/waba-cli/internal/config"
	"github.com/jjuanrivvera/waba-cli/internal/version"
)

// clientForSite builds a client for an explicitly chosen site, bypassing the usual
// --site resolution. Used by diagnostics and by cross-site commands, which need to talk to a
// site other than the active one.
func clientForSite(cmd *cobra.Command, o *globalOptions, site *config.Site) (*api.Client, *config.Site, error) {
	built, err := auth.Build(site, auth.NewStore())
	if err != nil {
		return nil, nil, err
	}
	hosts := &auth.Hosts{BaseURL: site.BaseURL, Method: site.AuthMethod, CloudID: site.CloudID}
	opts := []api.Option{
		api.WithAuthenticator(built.Authenticator),
		api.WithRateLimit(o.rps),
		api.WithUserAgent("atlassian-cli/" + version.Get().Version),
		api.WithShowToken(o.showToken),
		api.WithTimeout(o.timeout),
	}
	if o.dryRun {
		opts = append(opts, api.WithDryRun(true, cmd.OutOrStdout()))
	}
	if o.verbose {
		opts = append(opts, api.WithVerbose(cmd.ErrOrStderr()))
	}
	return api.NewClient(hosts, opts...), site, nil
}

// resolveSite loads config and returns a named site, for commands that take a site as an
// argument (cross-site sync) rather than through --site.
func resolveSite(name string) (*config.Site, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return cfg.Resolve(name)
}

// urlValues is a terse constructor for query parameters: urlValues("limit", "1").
func urlValues(kv ...string) url.Values {
	v := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Set(kv[i], kv[i+1])
	}
	return v
}

// renderRawList renders a list of raw JSON items (nested collections that have no dedicated
// Go type), decoding them once so the table renderer sees plain maps.
func (o *globalOptions) renderRawList(cmd *cobra.Command, items []json.RawMessage, columns []string, idField string) error {
	rows := make([]any, 0, len(items))
	for _, raw := range items {
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			continue
		}
		rows = append(rows, v)
	}
	return o.renderList(cmd, rows, columns, idField)
}

// readFileForFlag reads a file named directly on the command line by the user.
//
// No path confinement here on purpose: the user typed the path themselves, which is not the
// confused-deputy case that confinement protects against. Reads driven by *data* (a $ref
// inside an imported document) would need confinement; this does not.
func readFileForFlag(path string) ([]byte, error) {
	b, err := os.ReadFile(path) // #nosec G304 -- path supplied directly by the user on the CLI
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return b, nil
}

// jsonUnmarshalQuiet decodes without wrapping the error, for callers that treat a decode
// failure as "this optional field is absent" rather than as a fatal condition.
func jsonUnmarshalQuiet(raw []byte, out any) error { return json.Unmarshal(raw, out) }
