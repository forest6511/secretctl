---
title: Available Tools
description: MCP tools provided by secretctl.
sidebar_position: 4
---

# Available Tools

secretctl provides seven MCP tools for AI agents to work with secrets securely.

## Overview

| Tool | Description |
|------|-------------|
| `secret_list` | List secret keys with metadata (no values) |
| `secret_exists` | Check if a secret exists with metadata |
| `secret_get_masked` | Get masked secret value (e.g., `****WXYZ`) |
| `secret_run` | Execute command with secrets as environment variables |
| `secret_list_fields` | List field names for multi-field secrets (no values) |
| `secret_get_field` | Get non-sensitive field values only |
| `secret_run_with_bindings` | Execute with predefined environment bindings |

## secret_list

List all secret keys with metadata. Does **not** return secret values.

**Input Schema:**

```json
{
  "tag": "optional tag filter",
  "expiring_within": "optional expiration filter (e.g., '7d', '30d')"
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
  ]
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
  "key": "aws/access_key",
  "tags": ["aws", "prod"],
  "has_url": true,
  "has_notes": false,
  "expires_at": "2025-12-31T00:00:00Z",
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-06-15T10:30:00Z"
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
  "value_length": 20
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

## secret_list_fields

List all field names and metadata for a multi-field secret. Does **not** return field values.

**Input Schema:**

```json
{
  "key": "database/production"
}
```

**Example Response:**

```json
{
  "key": "database/production",
  "fields": [
    {
      "name": "host",
      "sensitive": false,
      "hint": "Database hostname",
      "kind": "string"
    },
    {
      "name": "password",
      "sensitive": true,
      "hint": "Database password"
    }
  ]
}
```

## secret_get_field

Get a specific field value from a multi-field secret. **Only non-sensitive fields can be retrieved** (Option D+ policy). Sensitive fields will be rejected.

**Input Schema:**

```json
{
  "key": "database/production",
  "field": "host"
}
```

**Example Response:**

```json
{
  "key": "database/production",
  "field": "host",
  "value": "db.example.com",
  "sensitive": false
}
```

**Error for Sensitive Field:**

```json
{
  "error": "field 'password' is marked as sensitive (Option D+ policy)"
}
```

## secret_run_with_bindings

Execute a command with environment variables injected based on the secret's predefined bindings. Each binding maps an environment variable name to a field.

**Input Schema:**

```json
{
  "key": "database/production",
  "command": "psql",
  "args": ["-c", "SELECT 1"],
  "timeout": "30s"
}
```

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `key` | string | yes | Secret key with bindings |
| `command` | string | yes | Command to execute |
| `args` | string[] | no | Command arguments |
| `timeout` | string | no | Execution timeout (default: 5m) |

**How Bindings Work:**

When a secret is created with bindings:

```bash
secretctl set database/production \
  --field host=db.example.com \
  --field port=5432 \
  --field password \
  --binding PGHOST=host \
  --binding PGPORT=port \
  --binding PGPASSWORD=password
```

Calling `secret_run_with_bindings` will inject:
- `PGHOST=db.example.com`
- `PGPORT=5432`
- `PGPASSWORD=<password value>`

**Example Response:**

```json
{
  "exit_code": 0,
  "stdout": " ?column? \n----------\n        1\n(1 row)\n",
  "stderr": "",
  "sanitized": true
}
```

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
