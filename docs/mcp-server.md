# MCP Server Guide

secretctl includes a Model Context Protocol (MCP) server for secure integration with AI coding assistants.

## Overview

The MCP server enables AI assistants like Claude Code, Codex CLI, and other MCP-compatible tools to work with your secrets **without ever seeing the actual secret values**.

### Security Design: Option D+

secretctl follows the "Access Without Exposure" principle:

- AI agents **never** receive plaintext secrets
- Secrets are injected as environment variables at runtime
- Command output is automatically sanitized to redact any leaked secrets
- A policy file controls which commands AI can execute

This design aligns with industry best practices from 1Password and HashiCorp Vault.

## Quick Start

### 1. Start the MCP Server

```bash
SECRETCTL_PASSWORD=your-master-password secretctl mcp-server
```

Or set the password via environment:
```bash
export SECRETCTL_PASSWORD=your-master-password
secretctl mcp-server
```

### 2. Configure Claude Code

Add to `~/.claude.json`:

```json
{
  "mcpServers": {
    "secretctl": {
      "command": "/path/to/secretctl",
      "args": ["mcp-server"],
      "env": {
        "SECRETCTL_PASSWORD": "your-master-password"
      }
    }
  }
}
```

### 3. Configure Codex CLI

Add to `~/.codex/config.yaml`:

```yaml
mcpServers:
  secretctl:
    command: /path/to/secretctl
    args:
      - mcp-server
    env:
      SECRETCTL_PASSWORD: your-master-password
```

## Available MCP Tools

### secret_list

List all secret keys with metadata. Does **not** return secret values.

**Input Schema:**
```json
{
  "tag": "optional tag filter",
  "prefix": "optional key prefix filter"
}
```

**Example Response:**
```json
{
  "secrets": [
    {
      "key": "aws/access_key",
      "tags": ["aws", "prod"],
      "has_url": true,
      "has_notes": false,
      "expires_at": "2025-12-31T00:00:00Z",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-06-15T10:30:00Z"
    }
  ],
  "total": 1
}
```

### secret_exists

Check if a secret key exists and return its metadata.

**Input Schema:**
```json
{
  "key": "aws/access_key"
}
```

**Example Response:**
```json
{
  "exists": true,
  "metadata": {
    "key": "aws/access_key",
    "tags": ["aws", "prod"],
    "has_url": true,
    "has_notes": false,
    "expires_at": "2025-12-31T00:00:00Z"
  }
}
```

### secret_get_masked

Get a masked version of a secret value. Useful for verification.

**Input Schema:**
```json
{
  "key": "aws/access_key"
}
```

**Example Response:**
```json
{
  "key": "aws/access_key",
  "masked_value": "****WXYZ",
  "length": 20
}
```

### secret_run

Execute a command with secrets injected as environment variables.

**Input Schema:**
```json
{
  "command": "aws",
  "args": ["s3", "ls"],
  "keys": ["aws/access_key", "aws/secret_key"],
  "timeout": "30s",
  "env_prefix": "AWS_"
}
```

**Features:**
- Secrets are injected as environment variables (e.g., `AWS_ACCESS_KEY`)
- Output is automatically sanitized to replace any leaked secrets with `[REDACTED:key]`
- Requires policy approval (see Policy Configuration below)
- Maximum 5 concurrent executions

**Example Response:**
```json
{
  "exit_code": 0,
  "stdout": "2024-01-15 mybucket\n2024-02-20 myotherbucket",
  "stderr": "",
  "sanitized": true
}
```

## Policy Configuration

The MCP server uses a policy file to control which commands AI can execute.

### Policy File Location

```
~/.secretctl/mcp-policy.yaml
```

### Policy Format

```yaml
version: 1

# Default action when command is not explicitly allowed
# Options: deny (recommended), allow
default_action: deny

# Commands that are allowed to be executed
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

  # Custom scripts
  - /path/to/deploy.sh

# Commands that are explicitly denied (overrides allowed)
denied_commands:
  - env        # Can leak all environment variables
  - printenv   # Can leak all environment variables
  - set        # Can leak shell state
  - export     # Can leak all exports
```

### Security Considerations

1. **deny-by-default**: Always use `default_action: deny`
2. **Explicit allowlist**: Only allow commands you trust
3. **Avoid shell built-ins**: Commands like `env`, `set`, `export` can leak secrets
4. **Use absolute paths**: For custom scripts, use full paths

### Commands Blocked by Default

Even if `default_action: allow`, these commands are always blocked:
- `env`
- `printenv`
- `set`
- `export`

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SECRETCTL_PASSWORD` | Master password for vault | (required) |
| `SECRETCTL_VAULT_DIR` | Custom vault directory | `~/.secretctl` |

## Troubleshooting

### Server won't start

**Error: "no password provided"**
```bash
# Set the password environment variable
export SECRETCTL_PASSWORD=your-master-password
secretctl mcp-server
```

**Error: "failed to unlock vault"**
- Check that the password is correct
- Verify the vault exists (`secretctl list` works)

### Commands are denied

**Error: "command not allowed by policy"**
1. Check `~/.secretctl/mcp-policy.yaml` exists
2. Add the command to `allowed_commands`
3. Restart the MCP server

### Output not sanitized

Output sanitization uses exact string matching. It does **not** detect:
- Base64-encoded secrets
- Hex-encoded secrets
- Partial string matches

## Best Practices

1. **Use strong master password**: The MCP server requires your master password
2. **Limit allowed commands**: Only allow commands you actually need
3. **Review policy regularly**: Audit your allowed commands list
4. **Use key prefixes**: Organize secrets with prefixes (e.g., `aws/`, `db/`)
5. **Set expirations**: Use `--expires` when setting sensitive secrets
6. **Monitor audit logs**: Check `secretctl audit list` for unusual activity

## Technical Details

### Protocol

- JSON-RPC 2.0 over stdio
- Compatible with MCP specification 2024-11-05

### Concurrency

- Maximum 5 concurrent `secret_run` executions
- Additional requests are queued

### Performance Targets

| Operation | p50 | p99 |
|-----------|-----|-----|
| secret_list | < 50ms | < 200ms |
| secret_exists | < 10ms | < 50ms |
| secret_get_masked | < 10ms | < 50ms |
| secret_run (startup) | < 100ms | < 500ms |
| MCP initialization | < 500ms | < 2s |

## Related Documentation

- [README.md](../README.md) - Project overview
- [SECURITY.md](../SECURITY.md) - Security policy
