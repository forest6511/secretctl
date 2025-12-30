---
title: For Developers
description: Get started with secretctl for AI-assisted development workflows.
sidebar_position: 2
---

# Getting Started: For Developers

This guide is for developers who want to integrate secretctl with AI coding assistants like Claude Code, automate secret injection in CI/CD, or use the MCP server for programmatic access.

## What You'll Learn

- Set up secretctl with Claude Code in 5 minutes
- Use the MCP server for AI-safe secret access
- Automate secret injection in your development workflow

## Prerequisites

- macOS, Linux, or Windows
- Terminal access
- Claude Code or similar AI coding assistant (optional)

## Step 1: Install secretctl

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-darwin-arm64
chmod +x secretctl-darwin-arm64
sudo mv secretctl-darwin-arm64 /usr/local/bin/secretctl

# macOS (Intel)
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-darwin-amd64
chmod +x secretctl-darwin-amd64
sudo mv secretctl-darwin-amd64 /usr/local/bin/secretctl

# Linux (x86_64)
curl -LO https://github.com/forest6511/secretctl/releases/latest/download/secretctl-linux-amd64
chmod +x secretctl-linux-amd64
sudo mv secretctl-linux-amd64 /usr/local/bin/secretctl

# Windows - Download from GitHub Releases
# https://github.com/forest6511/secretctl/releases/latest/download/secretctl-windows-amd64.exe
```

Verify installation:

```bash
secretctl --version
```

## Step 2: Initialize Your Vault

```bash
secretctl init
```

You'll be prompted to create a master password. Choose a strong password - this protects all your secrets.

:::tip Password Requirements
- Minimum 8 characters (required)
- 12+ characters recommended for strong security
- Mix of uppercase, lowercase, numbers, and symbols recommended
:::

## Step 3: Add Your API Keys

```bash
# OpenAI API key
echo "sk-proj-..." | secretctl set OPENAI_API_KEY

# AWS credentials
echo "AKIAIOSFODNN7EXAMPLE" | secretctl set aws/access_key_id
echo "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" | secretctl set aws/secret_access_key

# Database credentials
echo "postgres://user:pass@localhost:5432/db" | secretctl set db/connection_string

# With metadata
echo "ghp_xxxx" | secretctl set github/token \
  --notes "Personal access token for CI" \
  --tags "ci,github" \
  --expires "2025-12-31"
```

## Step 4: Configure Claude Code (MCP Integration)

Add to your Claude Code settings (`~/.config/claude-code/settings.json` or VS Code settings):

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "secretctl",
      "args": ["mcp-server"],
      "env": {
        "SECRETCTL_PASSWORD": "your-master-password"
      }
    }
  }
}
```

:::warning Security Note
The password is read once at startup and immediately cleared from the environment. Consider using a wrapper script or your shell's secure environment variable handling to avoid storing the password in config files.
:::

## Step 5: Use with Claude Code

Once configured, Claude Code can:

| Capability | Example |
|------------|---------|
| List your secrets | "What API keys do I have stored?" |
| Check if a key exists | "Do I have an OpenAI API key?" |
| Run commands with secrets | "Deploy to AWS using my credentials" |
| Get masked values | "Show me my GitHub token (masked)" |

**What Claude Code CANNOT do** (by design):
- Read plaintext secret values
- Modify or delete secrets
- Export your vault

This is the **Option D+** security model - AI agents can use your secrets without ever seeing them.

## Step 6: Run Commands with Secrets

Use `secretctl run` to inject secrets as environment variables:

```bash
# Run AWS CLI with credentials
secretctl run -k "aws/*" -- aws s3 ls

# Run with specific keys
secretctl run -k OPENAI_API_KEY -k ANTHROPIC_API_KEY -- python my_script.py

# Use environment aliases
secretctl run --env dev -k "db/*" -- ./migrate.sh
```

## Development Workflow Examples

### CI/CD Integration

```yaml
# GitHub Actions example
- name: Deploy
  env:
    SECRETCTL_PASSWORD: ${{ secrets.SECRETCTL_PASSWORD }}
  run: |
    secretctl run -k "aws/*" -- ./deploy.sh
```

### Local Development

```bash
# Start your dev server with all API keys
secretctl run -k "api/*" -- npm run dev

# Or export to .env for frameworks that need it
secretctl export --format env > .env
```

### Backup Your Vault

```bash
# Create encrypted backup
secretctl backup -o ~/backup/secrets-$(date +%Y%m%d).enc

# Restore when needed
secretctl restore ~/backup/secrets-20241224.enc --dry-run
```

## Next Steps

- [MCP Security Model](/docs/guides/mcp/security-model) - Understand how Option D+ keeps your secrets safe
- [Available MCP Tools](/docs/guides/mcp/available-tools) - Complete MCP tool reference
- [Environment Aliases](/docs/guides/mcp/env-aliases) - Manage dev/staging/prod environments
- [CLI Reference](/docs/reference/cli-commands) - Full CLI command documentation

## Troubleshooting

### Claude Code doesn't see secretctl

1. Verify the binary path: `which secretctl`
2. Use absolute path in config: `/usr/local/bin/secretctl`
3. Restart Claude Code after config changes

### "vault not initialized" error

Run `secretctl init` first, or check `SECRETCTL_VAULT_DIR` environment variable.

### MCP server connection issues

Check logs:
```bash
secretctl mcp-server 2>&1 | head -20
```
