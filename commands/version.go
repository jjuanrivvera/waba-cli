package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/output"
	"github.com/jjuanrivvera/waba-cli/internal/version"
)

func newVersionCmd(o *globalOptions) *cobra.Command {
	var (
		asJSON bool
		check  bool
	)
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := version.Get()
			if check {
				return runVersionCheck(cmd, info, asJSON)
			}
			if asJSON {
				// --json is explicit: it must win over the -o default (table) rather than
				// quietly rendering a table because nobody passed -o json as well.
				r := o.renderer(cmd, nil)
				r.Format = output.FormatJSON
				return r.Render(info)
			}
			fmt.Fprintln(cmd.OutOrStdout(), info.String())
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print as JSON")
	cmd.Flags().BoolVar(&check, "check", false, "check GitHub for a newer release")
	annotate(cmd, kindRead)
	cmd.Annotations["atlassianLocal"] = "true"
	return cmd
}

// latestReleaseURL is the GitHub API endpoint for the newest published release.
const latestReleaseURL = "https://api.github.com/repos/jjuanrivvera/atlassian-cli/releases/latest"

func runVersionCheck(cmd *cobra.Command, info version.Info, asJSON bool) error {
	latest, err := fetchLatestVersion(cmd.Context(), nil)
	if err != nil {
		return err
	}
	result := struct {
		Current  string `json:"current"`
		Latest   string `json:"latest"`
		Outdated bool   `json:"outdated"`
	}{Current: info.Version, Latest: latest, Outdated: latest != "" && latest != info.Version}

	if asJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
	if result.Outdated {
		fmt.Fprintf(cmd.OutOrStdout(), "atlassian %s (latest: %s) — run `atlassian update`\n", info.Version, latest)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "atlassian %s is up to date\n", info.Version)
	}
	return nil
}

// fetchLatestVersion reads the newest tag from GitHub. The client is injectable so tests
// drive it against httptest rather than the network.
func fetchLatestVersion(ctx context.Context, httpClient *http.Client) (string, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseCheckURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("check for updates: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("check for updates: %s", resp.Status)
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("check for updates: %w", err)
	}
	return rel.TagName, nil
}

// latestReleaseCheckURL is a variable so tests can point it at a local server.
var latestReleaseCheckURL = latestReleaseURL
