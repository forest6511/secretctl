package main

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script for your shell",
	Long: `To load completions:

Bash:
  $ source <(secretctl completion bash)

  # To load for each session (Linux):
  $ secretctl completion bash > ~/.local/share/bash-completion/completions/secretctl

  # To load for each session (macOS with Homebrew):
  $ secretctl completion bash > $(brew --prefix)/etc/bash_completion.d/secretctl

Zsh:
  # Ensure completion is enabled:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # Generate completion:
  $ secretctl completion zsh > ~/.zsh/completions/_secretctl
  # (create ~/.zsh/completions if needed, add to fpath in .zshrc)

Fish:
  $ secretctl completion fish > ~/.config/fish/completions/secretctl.fish

PowerShell:
  PS> secretctl completion powershell >> $PROFILE

Dynamic completion (secret keys):
  Set SECRETCTL_COMPLETION_ENABLED=1 to enable secret key completion.
  Note: Vault must be unlocked for this to work.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)

	// Register dynamic completion functions for commands
	registerCompletionFunctions()
}
