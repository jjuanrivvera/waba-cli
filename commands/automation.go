package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/waba-cli/internal/api"
	"github.com/jjuanrivvera/waba-cli/internal/config"
)

func init() {
	registerGroup("automation", "Conversational components: commands and ice breakers", nil, automationSpecs)
}

// errNothingToUpdate is shared by update verbs that build their payload from optional flags.
var errNothingToUpdate = errors.New("nothing to update — pass at least one flag")

func automationSpecs(o *globalOptions) []opSpec {
	return []opSpec{
		{
			Use: "get", Short: "Show the configured commands and ice breakers",
			Kind: kindRead, Args: cobra.NoArgs,
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				raw, err := c.Do(cmd.Context(), api.Request{Path: phone, Query: urlValues("fields", "conversational_automation")})
				if err != nil {
					return nil, err
				}
				return jsonMap(raw), nil
			},
		},
		{
			Use: "update", Short: "Configure commands and ice-breaker prompts",
			Long: "Commands appear when the user types “/”; ice breakers are tappable prompts on the\nfirst contact. Emojis and markdown are not supported in either.",
			Example: `  waba automation update \
    --command "cotizar:Recibe una cotización" --command "horario:Horario de atención" \
    --prompt "¿Cuánto cuesta una revisión?" --prompt "¿Atienden hoy?"`,
			Kind: kindWrite, Args: cobra.NoArgs,
			Flags: func(cmd *cobra.Command) {
				cmd.Flags().StringArray("command", nil, "bot command as name:description (repeatable, max 30)")
				cmd.Flags().StringArray("prompt", nil, "ice-breaker prompt text (repeatable, max 4, ≤80 chars)")
			},
			Run: func(cmd *cobra.Command, o *globalOptions, c *api.Client, acct *config.Account, args []string) (any, error) {
				phone, err := o.phoneNumberID(acct)
				if err != nil {
					return nil, err
				}
				cmds, _ := cmd.Flags().GetStringArray("command")
				prompts, _ := cmd.Flags().GetStringArray("prompt")
				if len(cmds) == 0 && len(prompts) == 0 {
					return nil, errNothingToUpdate
				}
				// enable_welcome_message is deliberately never sent — Meta removed it from
				// this endpoint (DECISIONS.md #9).
				body := map[string]any{}
				if len(cmds) > 0 {
					list := make([]map[string]string, len(cmds))
					for i, def := range cmds {
						name, desc, ok := strings.Cut(def, ":")
						if !ok {
							return nil, fmt.Errorf("--command %q is not name:description", def)
						}
						list[i] = map[string]string{"command_name": name, "command_description": desc}
					}
					body["commands"] = list
				}
				if len(prompts) > 0 {
					body["prompts"] = prompts
				}
				var out api.SuccessResult
				if err := c.PostJSON(cmd.Context(), phone+"/conversational_automation", body, &out); err != nil {
					return nil, err
				}
				o.noteWrite(cmd.ErrOrStderr(), "conversational components updated")
				return nil, nil
			},
		},
	}
}
