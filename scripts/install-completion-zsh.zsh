#!/bin/zsh
# Install zsh completion for secretctl
set -e

# Check if secretctl is available
if ! command -v secretctl &> /dev/null; then
    echo "Error: secretctl is not installed or not in PATH"
    exit 1
fi

# User-local completion directory
COMPLETION_DIR="$HOME/.zsh/completions"
mkdir -p "$COMPLETION_DIR"

# Generate completion
secretctl completion zsh > "$COMPLETION_DIR/_secretctl"

# Check if fpath includes this directory
if [[ ! " ${fpath[*]} " =~ " $COMPLETION_DIR " ]]; then
    echo ""
    echo "Add this to your ~/.zshrc:"
    echo '  fpath=(~/.zsh/completions $fpath)'
    echo '  autoload -U compinit && compinit'
fi

echo "Zsh completion installed to: $COMPLETION_DIR/_secretctl"
echo "Restart your shell for changes to take effect."
