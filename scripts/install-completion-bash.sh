#!/bin/bash
# Install bash completion for secretctl
set -e

# Check if secretctl is available
if ! command -v secretctl &> /dev/null; then
    echo "Error: secretctl is not installed or not in PATH"
    exit 1
fi

# User-local completion directory (no sudo required)
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS with Homebrew
    if command -v brew &> /dev/null; then
        COMPLETION_DIR="$(brew --prefix)/etc/bash_completion.d"
    else
        COMPLETION_DIR="$HOME/.local/share/bash-completion/completions"
    fi
else
    # Linux user directory
    COMPLETION_DIR="$HOME/.local/share/bash-completion/completions"
fi

mkdir -p "$COMPLETION_DIR"
secretctl completion bash > "$COMPLETION_DIR/secretctl"

echo "Bash completion installed to: $COMPLETION_DIR/secretctl"
echo "Restart your shell or run: source $COMPLETION_DIR/secretctl"
