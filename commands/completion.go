package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	registerMeta(func(root *cobra.Command, o *globalOptions) {
		root.AddCommand(newCompletionCmd(root))
	})
}

func newCompletionCmd(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate a shell completion script",
		Long: strings.TrimSpace(`
Generate a shell completion script.

Completion is worth installing here: it completes site names, output formats, column names
and — most usefully — the 1,143 operation ids that 'atlassian op call' accepts.

  bash:  atlassian completion bash > /etc/bash_completion.d/atlassian
  zsh:   atlassian completion zsh > "${fpath[1]}/_atlassian"
  fish:  atlassian completion fish > ~/.config/fish/completions/atlassian.fish
  pwsh:  atlassian completion powershell | Out-String | Invoke-Expression`),
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Write to the command's own stream, not os.Stdout, so tests can capture it with
			// cmd.SetOut instead of hijacking the process stdout (which deadlocks on a pipe
			// larger than the OS buffer — completion scripts are comfortably large enough).
			out := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(out, true)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(out)
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}
	annotate(cmd, kindRead)
	cmd.Annotations["atlassianLocal"] = "true"
	return cmd
}
