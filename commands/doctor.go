package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/auth"
	"github.com/jjuanrivvera/waba-cli/internal/config"
	"github.com/jjuanrivvera/waba-cli/internal/version"
)

func init() {
	registerMeta(func(root *cobra.Command, o *globalOptions) { root.AddCommand(newDoctorCmd(o)) })
}

type doctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

func newDoctorCmd(o *globalOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose configuration, credentials and connectivity",
		Long: `Runs the checks that explain 95% of "it doesn't work": config present, an account
resolvable, a token stored, the Graph API reachable, the token valid, and the configured
WABA / phone number accessible with it. Exits non-zero when anything fails.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var checks []doctorCheck
			add := func(name string, ok bool, detail string) {
				checks = append(checks, doctorCheck{Name: name, OK: ok, Detail: detail})
			}

			cfg, err := config.Load()
			if err != nil {
				add("config readable", false, err.Error())
			} else {
				add("config readable", true, cfg.FilePath())
			}

			var acct *config.Account
			if cfg != nil {
				acct, err = cfg.Resolve(o.account)
				if err != nil {
					add("account resolvable", false, err.Error())
				} else {
					add("account resolvable", true, acct.Name+" ("+acct.BaseURL+", "+acct.GraphVersion+")")
				}
			}

			store := auth.NewStore()
			add("credential backend", true, store.Backend())

			var token string
			if acct != nil {
				token, err = auth.ResolveToken(store, acct.Name)
				if err != nil {
					add("token stored", false, err.Error())
				} else {
					add("token stored", true, "")
				}
			}

			if acct != nil {
				// Reachability first, unauthenticated: separates "no network / wrong URL"
				// from "bad token", which need different fixes.
				reachClient := &http.Client{Timeout: 10 * time.Second}
				resp, err := reachClient.Get(acct.BaseURL + "/" + acct.GraphVersion + "/")
				if err != nil {
					add("graph api reachable", false, err.Error())
				} else {
					_ = resp.Body.Close()
					add("graph api reachable", true, resp.Status)
					// A large clock skew breaks OAuth-adjacent signatures and confuses
					// analytics windows; the Date header is a free sanity check.
					if d := resp.Header.Get("Date"); d != "" {
						if serverTime, perr := http.ParseTime(d); perr == nil {
							skew := time.Since(serverTime)
							if skew < 0 {
								skew = -skew
							}
							add("clock skew < 5m", skew < 5*time.Minute, skew.Round(time.Second).String())
						}
					}
				}
			}

			if acct != nil && token != "" {
				client := api.NewClient(acct.BaseURL, acct.GraphVersion,
					api.WithAuthenticator(&auth.Bearer{Token: token}), api.WithRateLimit(0), api.WithTimeout(o.timeout))

				if ident, err := verifyToken(cmd, o, acct, token); err != nil {
					add("token valid", false, err.Error())
				} else {
					add("token valid", true, ident)
				}

				if acct.WABAID != "" {
					var waba struct {
						ID   api.ID `json:"id"`
						Name string `json:"name"`
					}
					if err := client.GetJSON(cmd.Context(), acct.WABAID, urlValues("fields", "id,name"), &waba); err != nil {
						add("waba accessible", false, err.Error())
					} else {
						add("waba accessible", true, waba.Name)
					}
				}
				if acct.PhoneNumberID != "" {
					var phone struct {
						Display string `json:"display_phone_number"`
						Status  string `json:"status"`
					}
					if err := client.GetJSON(cmd.Context(), acct.PhoneNumberID, urlValues("fields", "display_phone_number,status"), &phone); err != nil {
						add("phone number accessible", false, err.Error())
					} else {
						add("phone number accessible", true, fmt.Sprintf("%s (%s)", phone.Display, phone.Status))
					}
				}
			}

			add("version", true, version.Get().String())

			failed := 0
			for _, c := range checks {
				if !c.OK {
					failed++
				}
			}

			if asJSON || o.output == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(map[string]any{"checks": checks, "ok": failed == 0}); err != nil {
					return err
				}
			} else {
				for _, c := range checks {
					mark := "✓"
					if !c.OK {
						mark = "✗"
					}
					line := fmt.Sprintf("%s %s", mark, c.Name)
					if c.Detail != "" {
						line += " — " + c.Detail
					}
					fmt.Fprintln(cmd.OutOrStdout(), line)
				}
			}
			if failed > 0 {
				return fmt.Errorf("%d check(s) failed", failed)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Annotations = map[string]string{"wabaLocal": "true"}
	return cmd
}
