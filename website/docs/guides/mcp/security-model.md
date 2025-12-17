---
title: Security Model
description: How Option D+ keeps secrets safe from AI agents.
sidebar_position: 2
---

# Security Model (Option D+)

secretctl follows the "Access Without Exposure" principle, ensuring AI agents can use your secrets without ever seeing them.

## Core Principles

1. **AI agents never receive plaintext secrets**
2. **Secrets are injected as environment variables at runtime**
3. **Command output is automatically sanitized to redact leaked secrets**
4. **A policy file controls which commands AI can execute**

This design aligns with industry best practices for secrets management.

## How It Works

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────┐
│  AI Agent   │────▶│ secretctl MCP    │────▶│   Command   │
│ (Claude)    │     │ Server           │     │ (aws, etc)  │
└─────────────┘     └──────────────────┘     └─────────────┘
       │                    │                       │
       │  "Run aws s3 ls    │  Inject secrets      │
       │   with aws/*"      │  as env vars         │
       │                    │                       │
       ▼                    ▼                       ▼
   Never sees          Manages secrets         Receives env
   secret values       and policy             vars securely
```

### What AI Agents Can Do

| Capability | Description |
|------------|-------------|
| List secrets | See key names and metadata (not values) |
| Check existence | Verify if a secret key exists |
| Get masked values | See last 4 characters only (`****WXYZ`) |
| Run commands | Execute allowed commands with secrets injected |

### What AI Agents Cannot Do

| Restriction | Reason |
|-------------|--------|
| Read plaintext | No `secret_get` tool exists |
| Access blocked commands | Policy enforcement |
| Bypass output sanitization | Automatic redaction |
| Run arbitrary commands | Allowlist-only policy |

## Policy Configuration

### Deny-by-Default

Always use `default_action: deny` in your policy:

```yaml
version: 1
default_action: deny  # Recommended
allowed_commands:
  - aws
  - gcloud
  - kubectl
```

### Commands Blocked by Default

Even with `default_action: allow`, these commands are always blocked:

- `env` - Can leak all environment variables
- `printenv` - Can leak all environment variables
- `set` - Can leak shell state
- `export` - Can leak all exports

### Denied Commands

Explicitly deny dangerous commands:

```yaml
denied_commands:
  - rm
  - dd
  - mkfs
```

## Output Sanitization

When a command is executed via `secret_run`, the output is automatically scanned for secret values. Any matches are replaced with `[REDACTED:key]`.

**Example:**

If `aws/secret_key` contains `AKIAIOSFODNN7EXAMPLE`:

```
# Original output
Access key: AKIAIOSFODNN7EXAMPLE

# Sanitized output
Access key: [REDACTED:aws/secret_key]
```

### Limitations

Output sanitization uses exact string matching. It does **not** detect:

- Base64-encoded secrets
- Hex-encoded secrets
- Partial string matches

## Best Practices

1. **Use strong master password** - The MCP server requires your master password
2. **Limit allowed commands** - Only allow commands you actually need
3. **Review policy regularly** - Audit your allowed commands list
4. **Use key prefixes** - Organize secrets with prefixes (e.g., `aws/`, `db/`)
5. **Set expirations** - Use `--expires` when setting sensitive secrets
6. **Monitor audit logs** - Check `secretctl audit list` for unusual activity
