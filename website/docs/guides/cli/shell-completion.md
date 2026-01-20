---
title: Shell Completion
description: Enable shell completion for secretctl commands.
sidebar_position: 5
---

# Shell Completion

secretctl provides shell completion for bash, zsh, fish, and PowerShell.

## Quick Install

### Bash

```bash
# Add to ~/.bashrc
echo 'source <(secretctl completion bash)' >> ~/.bashrc
source ~/.bashrc
```

### Zsh

```bash
# Add to ~/.zshrc
echo 'source <(secretctl completion zsh)' >> ~/.zshrc
source ~/.zshrc
```

### Fish

```bash
secretctl completion fish > ~/.config/fish/completions/secretctl.fish
```

### PowerShell

```powershell
# Add to your PowerShell profile
Add-Content $PROFILE 'secretctl completion powershell | Out-String | Invoke-Expression'
```

## What Gets Completed

Shell completion helps with:

- **Commands**: `secretctl <TAB>` shows available commands
- **Subcommands**: `secretctl completion <TAB>` shows shell options
- **Flags**: `secretctl get --<TAB>` shows available flags
- **Secret keys**: With dynamic completion enabled (see below)

## Dynamic Completion

By default, secretctl doesn't complete secret keys for security reasons. To enable dynamic secret key completion:

```bash
export SECRETCTL_COMPLETION_ENABLED=1
```

With dynamic completion enabled:

```bash
secretctl get <TAB>
# Shows available secret keys

secretctl delete <TAB>
# Shows available secret keys
```

:::caution Security Note
Dynamic completion requires the vault to be unlocked. If the vault is locked, key completion won't be available until you unlock it.
:::

### Enable Permanently

Add to your shell config:

```bash
# Bash (~/.bashrc) or Zsh (~/.zshrc)
export SECRETCTL_COMPLETION_ENABLED=1
source <(secretctl completion bash)  # or zsh
```

```fish
# Fish (~/.config/fish/config.fish)
set -gx SECRETCTL_COMPLETION_ENABLED 1
```

## Install Scripts

secretctl includes helper scripts for installing completion:

### Bash

```bash
# The install script adds completion to ~/.bash_completion.d/
./scripts/install-completion-bash.sh
```

### Zsh

```bash
# The install script adds completion to ~/.zsh/completions/
./scripts/install-completion-zsh.zsh
```

### Fish

```bash
# The install script adds completion to ~/.config/fish/completions/
./scripts/install-completion-fish.fish
```

## Verifying Installation

After installing, open a new terminal and test:

```bash
# Type secretctl and press Tab
secretctl <TAB>
# Should show: audit backup completion delete export generate get init list lock run set unlock

# Type a subcommand and press Tab for flags
secretctl get --<TAB>
# Should show available flags like --json, --silent
```

## Troubleshooting

### Completion Not Working

1. **Open a new terminal** - Completion scripts are loaded at shell startup
2. **Check the source command** - Ensure `source <(secretctl completion ...)` is in your shell config
3. **Verify secretctl is in PATH** - Run `which secretctl` to confirm

### Dynamic Completion Not Working

1. **Check the environment variable** - `echo $SECRETCTL_COMPLETION_ENABLED` should show `1`
2. **Unlock the vault** - Dynamic completion requires an unlocked vault
3. **Check vault status** - Run `secretctl list` to verify access

### Zsh Compinit Issues

If you get "command not found: compdef", add before the completion source:

```bash
autoload -Uz compinit && compinit
source <(secretctl completion zsh)
```

## Advanced: Custom Completion

The completion system uses Cobra's built-in completion. You can extend it by:

1. Forking secretctl
2. Adding `ValidArgsFunction` to commands in `cmd/secretctl/`
3. Rebuilding with `go build`

## Next Steps

- [CLI Commands Reference](/docs/reference/cli-commands) - Full command documentation
- [Running Commands](/docs/guides/cli/running-commands) - Use secrets with commands
