// Package commands is the cobra command tree.
//
// Command files register themselves from init() into a package-level queue, and NewRootCmd
// drains that queue onto a fresh root. Building a new root per call is what lets tests run
// commands in isolation: cobra flags are stateful and persist on a shared root, so reusing
// one leaks values between test cases.
package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/auth"
	"github.com/jjuanrivvera/waba-cli/internal/config"
	"github.com/jjuanrivvera/waba-cli/internal/output"
	"github.com/jjuanrivvera/waba-cli/internal/version"
)

// ProfileFlag is the multi-profile selector, named for what a profile actually is here: a
// WhatsApp Business Account plus its default phone number and app. `--profile` remains as a
// hidden alias so fleet muscle memory and existing scripts keep working.
const (
	ProfileFlag = "account"
	ProfileNoun = "account"
)

// globalOptions holds every persistent flag. One struct per root instance, so nothing is
// shared between tests.
type globalOptions struct {
	output    string
	account   string
	baseURL   string
	phoneID   string
	wabaIDF   string
	appIDF    string
	dryRun    bool
	showToken bool
	verbose   bool
	noColor   bool
	quiet     bool
	columns   []string
	jq        string
	rps       float64
	timeout   int
}

// registrars are resource/API command builders; metaRegistrars are setup and utility
// commands. They are kept apart so the MCP surface and the agent guard can reason about
// "commands that talk to the Graph API" versus "commands that configure this CLI".
var (
	registrars     []func(*cobra.Command, *globalOptions)
	metaRegistrars []func(*cobra.Command, *globalOptions)
)

func registerAPI(f func(*cobra.Command, *globalOptions))  { registrars = append(registrars, f) }
func registerMeta(f func(*cobra.Command, *globalOptions)) { metaRegistrars = append(metaRegistrars, f) }

// NewRootCmd builds a fresh command tree.
func NewRootCmd() *cobra.Command {
	opts := &globalOptions{output: output.FormatTable, rps: api.DefaultRPS, timeout: 60}

	root := &cobra.Command{
		Use:   "waba",
		Short: "WhatsApp Cloud API from the command line",
		Long: strings.TrimSpace(`
waba is a command-line client for Meta's WhatsApp Cloud API: send every message type,
manage media, phone numbers, message templates, the business profile, QR codes, WhatsApp
Flows, calling, groups and analytics — 102 documented operations, enumerated in
api-manifest.json.

Profiles ("accounts") bundle a WABA id, a default phone number id and an app id, so daily
use is just 'waba send text --to ... "hi"'. Tokens live in the OS keyring.`),
		Example: strings.TrimSpace(`
  # First-run setup: token, WABA id, default phone number
  waba init

  # Send messages
  waba send text --to 5730011122233 "Your order shipped!"
  waba send template --to 5730011122233 --name order_update --lang es_MX
  waba send image --to 5730011122233 --link https://example.com/cat.jpg

  # Templates
  waba templates list --status APPROVED
  waba templates create --data @welcome.json

  # Inspect the account
  waba phone list
  waba analytics messaging --start 2026-08-01 --end 2026-08-17 --granularity DAY

  # Anything else in the Graph API
  waba api GET me`),
		SilenceUsage:  true,
		SilenceErrors: true,
		// Cobra prints its own "unknown command" without suggestions unless asked.
		SuggestionsMinimumDistance: 2,
	}

	pf := root.PersistentFlags()
	pf.StringVarP(&opts.output, "output", "o", output.FormatTable,
		"output format: "+strings.Join(output.Formats, "|"))
	pf.StringVar(&opts.account, ProfileFlag, "", "named "+ProfileNoun+" to use")
	// Both flags target the same variable, so --account and the legacy --profile are
	// interchangeable; --profile is hidden to keep the help output honest about the name.
	pf.StringVar(&opts.account, "profile", "", "alias for --"+ProfileFlag)
	_ = pf.MarkHidden("profile")
	pf.StringVar(&opts.baseURL, "base-url", "", "override the Graph API base URL")
	pf.StringVar(&opts.phoneID, "phone-id", "", "business phone number id (overrides the account default)")
	pf.StringVar(&opts.wabaIDF, "waba-id", "", "WhatsApp Business Account id (overrides the account default)")
	pf.StringVar(&opts.appIDF, "app-id", "", "Meta app id (overrides the account default)")
	pf.BoolVar(&opts.dryRun, "dry-run", false, "print the equivalent curl command and send nothing")
	pf.BoolVar(&opts.showToken, "show-token", false, "do not redact credentials in --dry-run output")
	pf.BoolVarP(&opts.verbose, "verbose", "v", false, "trace requests to stderr")
	pf.BoolVar(&opts.noColor, "no-color", false, "disable colored output")
	pf.BoolVar(&opts.quiet, "quiet", false, "suppress notes and warnings")
	pf.StringSliceVar(&opts.columns, "columns", nil, "columns to show in table/csv output")
	pf.StringVar(&opts.jq, "jq", "", "filter the result through a gojq expression")
	pf.Float64Var(&opts.rps, "rps", api.DefaultRPS, "client-side request rate limit (requests/second)")
	pf.IntVar(&opts.timeout, "timeout", 60, "per-request timeout in seconds")

	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if err := output.ValidateFormat(opts.output); err != nil {
			return err
		}
		// NO_COLOR is honoured here rather than at each print site, so every command obeys it.
		if !output.ColorEnabled(cmd.OutOrStdout(), opts.noColor) {
			opts.noColor = true
		}
		return nil
	}

	root.AddCommand(newVersionCmd(opts))
	for _, r := range metaRegistrars {
		r(root, opts)
	}
	for _, r := range registrars {
		r(root, opts)
	}
	return root
}

// Execute builds and runs the tree. Errors are printed here, once, rather than by cobra.
func Execute(ctx context.Context) int {
	root := NewRootCmd()
	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		return 1
	}
	return 0
}

// clientFor resolves configuration and credentials into a ready API client.
//
// Every command that talks to the Graph API goes through here, so precedence
// (flag > env > config > default) is applied in exactly one place.
func (o *globalOptions) clientFor(cmd *cobra.Command) (*api.Client, *config.Account, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	acct, err := cfg.Resolve(o.account)
	if err != nil {
		return nil, nil, err
	}
	if o.baseURL != "" {
		if err := config.ValidateBaseURL(o.baseURL); err != nil {
			return nil, nil, err
		}
		clone := *acct
		clone.BaseURL = o.baseURL
		acct = &clone
	}

	token, err := auth.ResolveToken(auth.NewStore(), acct.Name)
	if err != nil {
		return nil, nil, err
	}

	clientOpts := []api.Option{
		api.WithAuthenticator(&auth.Bearer{Token: token}),
		api.WithRateLimit(o.rps),
		api.WithUserAgent("waba-cli/" + version.Get().Version),
		api.WithShowToken(o.showToken),
		api.WithTimeout(o.timeout),
	}
	if o.dryRun {
		clientOpts = append(clientOpts, api.WithDryRun(true, cmd.OutOrStdout()))
	}
	if o.verbose {
		clientOpts = append(clientOpts, api.WithVerbose(cmd.ErrOrStderr()))
	}
	return api.NewClient(acct.BaseURL, acct.GraphVersion, clientOpts...), acct, nil
}

// phoneID resolves the business phone number id: --phone-id > WABA_PHONE_NUMBER_ID (already
// layered into the account) > the account default.
func (o *globalOptions) phoneNumberID(acct *config.Account) (string, error) {
	if o.phoneID != "" {
		return o.phoneID, nil
	}
	if acct.PhoneNumberID != "" {
		return acct.PhoneNumberID, nil
	}
	return "", fmt.Errorf("no phone number id — pass --phone-id, set WABA_PHONE_NUMBER_ID, or store one with `waba init` (find ids with `waba phone list`)")
}

// wabaID resolves the WhatsApp Business Account id the same way.
func (o *globalOptions) wabaID(acct *config.Account) (string, error) {
	if o.wabaIDF != "" {
		return o.wabaIDF, nil
	}
	if acct.WABAID != "" {
		return acct.WABAID, nil
	}
	return "", fmt.Errorf("no WABA id — pass --waba-id, set WABA_WABA_ID, or store one with `waba init`")
}

// appID resolves the Meta app id (resumable uploads only).
func (o *globalOptions) appID(acct *config.Account) (string, error) {
	if o.appIDF != "" {
		return o.appIDF, nil
	}
	if acct.AppID != "" {
		return acct.AppID, nil
	}
	return "", fmt.Errorf("no app id — pass --app-id, set WABA_APP_ID, or store one with `waba init` (App Dashboard > App settings > Basic)")
}

// renderer builds the output renderer for a command, wired to that command's streams so
// tests can capture output with cmd.SetOut.
func (o *globalOptions) renderer(cmd *cobra.Command, preferred []string) *output.Renderer {
	r := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), o.output)
	r.Columns = o.columns
	r.Preferred = preferred
	r.NoColor = o.noColor
	r.Quiet = o.quiet
	return r
}

// render sends a value to stdout in the selected format, applying --jq first when present.
func (o *globalOptions) render(cmd *cobra.Command, v any, preferred []string) error {
	if o.jq != "" {
		filtered, err := applyJQ(o.jq, v)
		if err != nil {
			return err
		}
		v = filtered
	}
	return o.renderer(cmd, preferred).Render(v)
}

// noteWrite reports that a mutating command succeeded.
//
// It stays silent during a dry run: nothing was sent, so claiming "sent" would be a plain
// falsehood, and the printed curl is already the output the flag promised.
func (o *globalOptions) noteWrite(w io.Writer, format string, args ...any) {
	if o.dryRun {
		return
	}
	o.note(w, format, args...)
}

// note writes an advisory to stderr, keeping stdout clean for pipes.
func (o *globalOptions) note(w io.Writer, format string, args ...any) {
	if o.quiet {
		return
	}
	fmt.Fprintf(w, "note: "+format+"\n", args...)
}
