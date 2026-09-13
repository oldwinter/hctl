package cli

import (
	"github.com/spf13/cobra"

	"github.com/oldwinter/hctl/internal/exitcode"
)

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `To load completions:

  # bash
  source <(hctl completion bash)

  # zsh
  source <(hctl completion zsh)

  # fish
  hctl completion fish | source
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return exitcode.Errorf(exitcode.Usage, "completion bash|zsh|fish|powershell (example: hctl completion bash)")
			}
			return nil
		},
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return exitcode.Errorf(exitcode.Usage, "unknown shell %q (want bash|zsh|fish|powershell)", args[0])
			}
		},
	}
}
