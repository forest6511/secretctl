#!/bin/bash
# Demo recording script for asciinema (~30 seconds)
#
# Prerequisites:
#   brew install asciinema agg  # or: pip install asciinema && cargo install agg
#
# Setup (run BEFORE recording):
#   export SECRETCTL_VAULT_DIR="/tmp/secretctl-demo"
#   rm -rf "$SECRETCTL_VAULT_DIR"
#   echo "demo123456" | secretctl init  # Initialize with demo password
#
# Recording:
#   export SECRETCTL_PASSWORD="demo123456"
#   asciinema rec demo.cast -c "./scripts/demo-recording.sh"
#
# Convert to GIF:
#   agg demo.cast demo.gif --font-size 16 --theme monokai --cols 80 --rows 24
#
# Or use asciinema.org for SVG player

set -e

# Use demo vault (must be pre-initialized - see Setup above)
export SECRETCTL_VAULT_DIR="${SECRETCTL_VAULT_DIR:-/tmp/secretctl-demo}"

# Simple typing effect (no external dependencies)
slow_type() {
    for ((i=0; i<${#1}; i++)); do
        printf '%s' "${1:$i:1}"
        sleep 0.03
    done
    echo
}

run() {
    printf '\033[0;34m$ \033[0m'
    slow_type "$1"
    sleep 0.2
    eval "$1"
    sleep 1.2
}

clear
echo -e "\033[1;32m# secretctl - The simplest AI-ready secrets manager\033[0m"
echo
sleep 1.5

# Store a secret (use fake demo key - never record real secrets!)
run "echo 'sk-proj-abc123xyz' | secretctl set OPENAI_API_KEY"

# List secrets
run "secretctl list"

# Get the secret value
run "secretctl get OPENAI_API_KEY"

# Run command with secret injection (output is sanitized)
echo
echo -e "\033[1;33m# Secrets injected as env vars - output is sanitized:\033[0m"
sleep 1
run "secretctl run -k OPENAI_API_KEY -- sh -c 'echo My key is \$OPENAI_API_KEY'"

# Cleanup
run "secretctl delete OPENAI_API_KEY --force"

echo
echo -e "\033[1;32m# Local-first. AI-safe. Just works.\033[0m"
echo -e "\033[0;36mhttps://github.com/forest6511/secretctl\033[0m"
sleep 2
