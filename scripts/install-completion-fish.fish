#!/usr/bin/env fish
# Install fish completion for secretctl

# Check if secretctl is available
if not command -v secretctl &> /dev/null
    echo "Error: secretctl is not installed or not in PATH"
    exit 1
end

set COMPLETION_DIR "$HOME/.config/fish/completions"
mkdir -p "$COMPLETION_DIR"

secretctl completion fish > "$COMPLETION_DIR/secretctl.fish"

echo "Fish completion installed to: $COMPLETION_DIR/secretctl.fish"
echo "Restart your shell for changes to take effect."
