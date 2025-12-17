---
title: MCP Tools
description: Complete API reference for secretctl MCP server tools.
sidebar_position: 2
---

# MCP Tools Reference

Complete API reference for all MCP tools provided by the secretctl MCP server.

## Overview

The secretctl MCP server implements a security-first design where **AI agents never receive plaintext secrets**. This follows the "Option D+" architecture, aligned with 1Password's "Access Without Exposure" philosophy.

**Available Tools:**

| Tool | Description |
|------|-------------|
| `secret_list` | List secret keys with metadata (no values) |
| `secret_exists` | Check if a secret exists with metadata |
| `secret_get_masked` | Get masked secret value (e.g., `****WXYZ`) |
| `secret_run` | Execute command with secrets as environment variables |

---

## secret_list

List all secret keys with metadata. Returns key names, tags, expiration, and flags for notes/URL presence. Does NOT return secret values.

### Input Schema

```json
{
  "tag": "string (optional)",
  "expiring_within": "string (optional)"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tag` | string | No | Filter by tag |
| `expiring_within` | string | No | Filter by expiration (e.g., `7d`, `30d`) |

### Output Schema

```json
{
  "secrets": [
    {
      "key": "string",
      "tags": ["string"],
      "expires_at": "string (RFC 3339, optional)",
      "has_notes": "boolean",
      "has_url": "boolean",
      "created_at": "string (RFC 3339)",
      "updated_at": "string (RFC 3339)"
    }
  ]
}
```

### Examples

**List all secrets:**

```json
// Input
{}

// Output
{
  "secrets": [
    {
      "key": "AWS_ACCESS_KEY",
      "tags": ["aws", "prod"],
      "has_notes": false,
      "has_url": true,
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T10:30:00Z"
    },
    {
      "key": "DB_PASSWORD",
      "tags": ["db", "prod"],
      "expires_at": "2025-06-15T00:00:00Z",
      "has_notes": true,
      "has_url": false,
      "created_at": "2025-01-10T08:00:00Z",
      "updated_at": "2025-01-10T08:00:00Z"
    }
  ]
}
```

**Filter by tag:**

```json
// Input
{
  "tag": "prod"
}

// Output
{
  "secrets": [/* secrets with "prod" tag */]
}
```

**Filter by expiration:**

```json
// Input
{
  "expiring_within": "30d"
}

// Output
{
  "secrets": [/* secrets expiring within 30 days */]
}
```

---

## secret_exists

Check if a secret key exists and return its metadata. Does NOT return the secret value.

### Input Schema

```json
{
  "key": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `key` | string | Yes | The secret key to check |

### Output Schema

```json
{
  "exists": "boolean",
  "key": "string",
  "tags": ["string"],
  "expires_at": "string (RFC 3339, optional)",
  "has_notes": "boolean",
  "has_url": "boolean",
  "created_at": "string (RFC 3339, optional)",
  "updated_at": "string (RFC 3339, optional)"
}
```

### Examples

**Check existing secret:**

```json
// Input
{
  "key": "API_KEY"
}

// Output
{
  "exists": true,
  "key": "API_KEY",
  "tags": ["api", "prod"],
  "has_notes": true,
  "has_url": true,
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

**Check non-existent secret:**

```json
// Input
{
  "key": "NONEXISTENT_KEY"
}

// Output
{
  "exists": false,
  "key": "NONEXISTENT_KEY",
  "tags": null,
  "has_notes": false,
  "has_url": false
}
```

---

## secret_get_masked

Get a masked version of a secret value (e.g., `****WXYZ`). Useful for verifying secret format without exposing the actual value.

### Input Schema

```json
{
  "key": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `key` | string | Yes | The secret key to retrieve |

### Output Schema

```json
{
  "key": "string",
  "masked_value": "string",
  "value_length": "integer"
}
```

### Masking Behavior

The masking algorithm:
- Shows the last 4 characters of the secret
- Replaces all preceding characters with `*`
- For secrets shorter than 8 characters, shows only asterisks

| Secret Length | Masked Output |
|--------------|---------------|
| 1-7 chars | `*******` (all asterisks) |
| 8+ chars | `****WXYZ` (last 4 visible) |

### Examples

**Get masked value:**

```json
// Input
{
  "key": "API_KEY"
}

// Output (assuming API_KEY = "sk-abc123xyz789")
{
  "key": "API_KEY",
  "masked_value": "*********789",
  "value_length": 14
}
```

**Short secret:**

```json
// Input
{
  "key": "PIN"
}

// Output (assuming PIN = "1234")
{
  "key": "PIN",
  "masked_value": "****",
  "value_length": 4
}
```

---

## secret_run

Execute a command with specified secrets injected as environment variables. Output is automatically sanitized to prevent secret leakage. Requires policy approval.

### Input Schema

```json
{
  "keys": ["string"],
  "command": "string",
  "args": ["string"],
  "timeout": "string (optional)",
  "env_prefix": "string (optional)",
  "env": "string (optional)"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `keys` | string[] | Yes | Secret keys to inject (glob patterns supported) |
| `command` | string | Yes | Command to execute |
| `args` | string[] | No | Command arguments |
| `timeout` | string | No | Execution timeout (e.g., `30s`, `5m`). Default: `5m` |
| `env_prefix` | string | No | Prefix for environment variable names |
| `env` | string | No | Environment alias (e.g., `dev`, `staging`, `prod`) |

### Output Schema

```json
{
  "exit_code": "integer",
  "stdout": "string",
  "stderr": "string",
  "duration_ms": "integer",
  "sanitized": "boolean"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `exit_code` | integer | Command exit code (0 = success) |
| `stdout` | string | Standard output (sanitized) |
| `stderr` | string | Standard error (sanitized) |
| `duration_ms` | integer | Execution duration in milliseconds |
| `sanitized` | boolean | Whether output was sanitized |

### Key Pattern Syntax

The `keys` field supports glob patterns:

| Pattern | Matches |
|---------|---------|
| `API_KEY` | Exact match for `API_KEY` |
| `aws/*` | All keys under `aws/` (single level) |
| `db/*` | All keys under `db/` (single level) |

### Environment Variable Naming

Secret keys are transformed to environment variable names:

- `/` is replaced with `_`
- `-` is replaced with `_`
- Names are converted to UPPERCASE

| Secret Key | Environment Variable |
|------------|---------------------|
| `aws/access_key` | `AWS_ACCESS_KEY` |
| `db-password` | `DB_PASSWORD` |
| `api/prod/key` | `API_PROD_KEY` |

### Output Sanitization

All command output is scanned for secret values. Any matches are replaced with `[REDACTED:key]`:

```
Original: "Connected to database with password secret123"
Sanitized: "Connected to database with password [REDACTED:DB_PASSWORD]"
```

### Examples

**Run with single secret:**

```json
// Input
{
  "keys": ["API_KEY"],
  "command": "curl",
  "args": ["-H", "Authorization: Bearer $API_KEY", "https://api.example.com"]
}

// Output
{
  "exit_code": 0,
  "stdout": "{\"status\": \"ok\"}",
  "stderr": "",
  "duration_ms": 245,
  "sanitized": false
}
```

**Run with wildcard pattern:**

```json
// Input
{
  "keys": ["aws/*"],
  "command": "aws",
  "args": ["s3", "ls"]
}

// Output
{
  "exit_code": 0,
  "stdout": "2025-01-15 10:30:00 my-bucket\n",
  "stderr": "",
  "duration_ms": 1250,
  "sanitized": false
}
```

**Run with environment alias:**

```json
// Input
{
  "keys": ["db/*"],
  "command": "./deploy.sh",
  "env": "prod"
}

// Output
{
  "exit_code": 0,
  "stdout": "Deployment complete",
  "stderr": "",
  "duration_ms": 5000,
  "sanitized": false
}
```

**Run with prefix:**

```json
// Input
{
  "keys": ["API_KEY"],
  "command": "./app",
  "env_prefix": "MYAPP_"
}

// Environment variables:
// MYAPP_API_KEY=<value>

// Output
{
  "exit_code": 0,
  "stdout": "Application started",
  "stderr": "",
  "duration_ms": 100,
  "sanitized": false
}
```

---

## Policy Configuration

The `secret_run` tool requires policy approval. Create `~/.secretctl/mcp-policy.yaml`:

```yaml
version: 1
default_action: deny

# Commands that are always blocked (security)
# - env, printenv, set, export, cat /proc/*/environ

# User-defined denied commands
denied_commands: []

# Allowed commands (required when default_action is deny)
allowed_commands:
  - aws
  - gcloud
  - kubectl
  - curl
  - ./deploy.sh

# Environment aliases for key transformation
env_aliases:
  dev:
    - pattern: "db/*"
      target: "dev/db/*"
  prod:
    - pattern: "db/*"
      target: "prod/db/*"
```

### Policy Evaluation Order

1. **Default denied commands** (always blocked): `env`, `printenv`, `set`, `export`
2. **User-defined `denied_commands`**: Explicitly blocked commands
3. **User-defined `allowed_commands`**: Explicitly allowed commands
4. **`default_action`**: Fallback action (`allow` or `deny`)

### Security Requirements

The policy file must meet these requirements:
- File permissions: `0600` (owner read/write only)
- No symlinks (direct file only)
- Owned by current user

---

## Security Design

### Option D+ Architecture

The secretctl MCP server follows the "Option D+" security model:

| Tool | Plaintext Access | Purpose |
|------|-----------------|---------|
| `secret_list` | No | List keys and metadata only |
| `secret_exists` | No | Check existence and metadata |
| `secret_get_masked` | No | Verify format without exposure |
| `secret_run` | No* | Inject via environment variables |

\* Secrets are injected as environment variables into the child process. The AI agent never sees the plaintext values.

### Why No `secret_get`?

A `secret_get` tool that returns plaintext values is intentionally **not implemented**. This design choice aligns with:

- **1Password**: Explicitly refuses to expose raw credentials via MCP
- **HashiCorp Vault**: "Raw secrets never exposed" policy
- **Industry best practice**: Minimize secret exposure surface

### Output Sanitization

The `secret_run` tool automatically sanitizes command output to prevent accidental secret leakage:

- Exact string matching for secret values
- Replacement with `[REDACTED:key]` placeholder
- Applied to both stdout and stderr

**Limitations:**
- Base64 or hex-encoded secrets are not detected
- Partial matches are not detected
- Obfuscated or transformed values are not detected

---

## Error Handling

### Common Errors

| Error | Description |
|-------|-------------|
| `secret not found` | The requested secret key does not exist |
| `policy not found` | No MCP policy file exists |
| `command not allowed` | Command blocked by policy |
| `timeout exceeded` | Command execution exceeded timeout |

### Error Response Format

```json
{
  "error": {
    "code": -32000,
    "message": "secret not found: API_KEY"
  }
}
```

---

## Integration Example

Configure in Claude Code (`~/.claude.json`):

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

See [MCP Integration Guide](/docs/guides/mcp/) for detailed setup instructions.
