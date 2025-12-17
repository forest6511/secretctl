---
title: Available Tools
description: MCP tools provided by secretctl.
sidebar_position: 4
---

# Available Tools

secretctl provides four MCP tools for AI agents to work with secrets securely.

## secret_list

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

## secret_exists

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

## secret_get_masked

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

## secret_run

Execute a command with secrets injected as environment variables.

**Input Schema:**

```json
{
  "command": "aws",
  "args": ["s3", "ls"],
  "keys": ["aws/access_key", "aws/secret_key"],
  "timeout": "30s",
  "env_prefix": "AWS_",
  "env": "prod"
}
```

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `command` | string | yes | Command to execute |
| `args` | string[] | no | Command arguments |
| `keys` | string[] | yes | Secret key patterns (glob supported) |
| `timeout` | string | no | Execution timeout (default: 5m, max: 1h) |
| `env_prefix` | string | no | Prefix for environment variable names |
| `env` | string | no | Environment alias (e.g., "dev", "staging", "prod") |

**Features:**

- Secrets are injected as environment variables
- Output is automatically sanitized to replace any leaked secrets with `[REDACTED:key]`
- Requires policy approval
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

### Environment Variable Naming

Secret keys are transformed to environment variable names as follows:

1. Slashes (`/`) are replaced with underscores (`_`)
2. Hyphens (`-`) are replaced with underscores (`_`)
3. The result is converted to UPPERCASE
4. If `env_prefix` is specified, it is prepended

**Examples:**

| Secret Key | env_prefix | Environment Variable |
|------------|------------|---------------------|
| `aws/access_key` | (none) | `AWS_ACCESS_KEY` |
| `aws/access_key` | `MY_` | `MY_AWS_ACCESS_KEY` |
| `db-password` | (none) | `DB_PASSWORD` |
| `api/prod/key` | `APP_` | `APP_API_PROD_KEY` |

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
