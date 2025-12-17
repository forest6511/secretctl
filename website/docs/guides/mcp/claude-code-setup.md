---
title: Claude Code Setup
description: Set up secretctl with Claude Code.
sidebar_position: 3
---

# Claude Code Setup

This guide covers how to configure secretctl MCP server with Claude Code and other MCP-compatible AI tools.

## Prerequisites

1. secretctl installed and initialized (`secretctl init`)
2. At least one secret stored (`secretctl secret add`)
3. Policy file created (see below)

## Step 1: Create Policy File

Create `~/.secretctl/mcp-policy.yaml`:

```bash
mkdir -p ~/.secretctl
cat > ~/.secretctl/mcp-policy.yaml << 'EOF'
version: 1
default_action: deny
allowed_commands:
  # Cloud CLIs
  - aws
  - gcloud
  - az

  # Container tools
  - kubectl
  - docker
  - helm

  # Database tools
  - psql
  - mysql
  - mongosh

denied_commands:
  - env
  - printenv
  - set
  - export
EOF
```

## Step 2: Configure Claude Code

Add to `~/.claude.json`:

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "/path/to/secretctl",
      "args": ["mcp", "serve"],
      "env": {
        "SECRETCTL_PASSWORD": "your-master-password"
      }
    }
  }
}
```

:::caution
Replace `/path/to/secretctl` with the actual path to your secretctl binary.
You can find it with `which secretctl`.
:::

### Using Homebrew Installation

If installed via Homebrew:

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "/opt/homebrew/bin/secretctl",
      "args": ["mcp", "serve"],
      "env": {
        "SECRETCTL_PASSWORD": "your-master-password"
      }
    }
  }
}
```

## Step 3: Configure Codex CLI

For OpenAI Codex CLI, add to `~/.codex/config.yaml`:

```yaml
mcpServers:
  secretctl:
    command: /path/to/secretctl
    args:
      - mcp
      - serve
    env:
      SECRETCTL_PASSWORD: your-master-password
```

## Step 4: Test the Integration

1. Start Claude Code or your MCP client
2. Ask Claude to list your secrets:
   ```
   "List my available secrets"
   ```
3. Claude should respond with your secret keys (not values)

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SECRETCTL_PASSWORD` | Master password for vault | (required) |
| `SECRETCTL_VAULT_DIR` | Custom vault directory | `~/.secretctl` |

## Troubleshooting

### Server won't start

**Error: "no password provided"**

```bash
# Verify the password is set correctly
export SECRETCTL_PASSWORD=your-master-password
secretctl mcp serve
```

**Error: "failed to unlock vault"**

- Check that the password is correct
- Verify the vault exists (`secretctl list` works)

### Commands are denied

**Error: "command not allowed by policy"**

1. Check `~/.secretctl/mcp-policy.yaml` exists
2. Add the command to `allowed_commands`
3. Restart Claude Code to reload the MCP server

### Connection Issues

If Claude Code can't connect:

1. Check the secretctl path is correct
2. Verify secretctl is executable
3. Test manually: `secretctl mcp serve`

## Security Recommendations

1. **Don't commit secrets** - Never add your Claude config with passwords to git
2. **Use environment variables** - Consider using a shell wrapper for the password
3. **Limit permissions** - Only allow the commands you need
4. **Review regularly** - Check `secretctl audit list` for unusual activity

## Example Workflow

Once configured, you can ask Claude to:

```
"Run 'aws s3 ls' using my aws/* credentials"

"Deploy to Kubernetes using my k8s/prod/* secrets"

"Connect to the database with db/prod/password"
```

Claude will use `secret_run` to execute these commands with secrets injected as environment variables, without ever seeing the actual values.
