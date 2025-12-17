---
title: Configuration
description: Configuration options, environment variables, and file structure.
sidebar_position: 3
---

# Configuration Reference

Complete reference for secretctl configuration options, environment variables, and file structure.

## Environment Variables

### Vault Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SECRETCTL_VAULT_DIR` | Directory containing the vault files | `~/.secretctl` |
| `SECRETCTL_PASSWORD` | Master password for vault operations | (none) |

**Usage Examples:**

```bash
# Use a custom vault directory
export SECRETCTL_VAULT_DIR=/path/to/custom/vault
secretctl list

# Provide password via environment variable (non-interactive)
SECRETCTL_PASSWORD=mypassword secretctl get API_KEY

# MCP server uses SECRETCTL_PASSWORD for authentication
SECRETCTL_PASSWORD=mypassword secretctl mcp-server
```

### Security Notes

- `SECRETCTL_PASSWORD` is automatically cleared from the environment after reading
- Avoid setting `SECRETCTL_PASSWORD` in shell profiles or persistent environment
- For MCP server, use process-level environment variables in your MCP client configuration

---

## File Structure

### Vault Directory

The default vault directory is `~/.secretctl`. The structure is:

```
~/.secretctl/
├── vault.salt       # Cryptographic salt (16 bytes)
├── vault.meta       # Vault metadata (encrypted)
├── vault.db         # SQLite database (encrypted)
├── vault.lock       # Lock file for concurrent access
├── audit/           # Audit logs directory
│   └── *.jsonl      # JSON Lines audit log files
└── mcp-policy.yaml  # MCP server policy (optional)
```

### File Permissions

All files are created with secure permissions:

| File/Directory | Permissions | Description |
|---------------|-------------|-------------|
| `~/.secretctl/` | `0700` | Directory accessible only by owner |
| `vault.salt` | `0600` | Salt file (owner read/write only) |
| `vault.meta` | `0600` | Metadata file (owner read/write only) |
| `vault.db` | `0600` | Database file (owner read/write only) |
| `mcp-policy.yaml` | `0600` | Policy file (required for MCP server) |
| `audit/` | `0700` | Audit logs directory |

**Important:** The MCP policy file must have `0600` permissions and be owned by the current user. Symlinks are not allowed for security reasons.

---

## MCP Policy Configuration

Create `~/.secretctl/mcp-policy.yaml` to configure the MCP server:

```yaml
version: 1
default_action: deny

# Commands that are always blocked (hardcoded)
# - env, printenv, set, export, cat /proc/*/environ

# User-defined denied commands (checked first)
denied_commands:
  - rm
  - mv
  - sudo

# Allowed commands (checked second)
allowed_commands:
  - aws
  - gcloud
  - kubectl
  - curl
  - wget
  - ./deploy.sh

# Environment aliases for key transformation
env_aliases:
  dev:
    - pattern: "db/*"
      target: "dev/db/*"
    - pattern: "api/*"
      target: "dev/api/*"
  staging:
    - pattern: "db/*"
      target: "staging/db/*"
    - pattern: "api/*"
      target: "staging/api/*"
  prod:
    - pattern: "db/*"
      target: "prod/db/*"
    - pattern: "api/*"
      target: "prod/api/*"
```

### Policy Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | integer | Yes | Policy version (must be `1`) |
| `default_action` | string | No | Default action: `allow` or `deny` (default: `deny`) |
| `denied_commands` | string[] | No | Commands to always block |
| `allowed_commands` | string[] | No | Commands to allow |
| `env_aliases` | map | No | Environment alias mappings |

### Policy Evaluation Order

1. **Hardcoded denies**: `env`, `printenv`, `set`, `export`, `cat /proc/*/environ`
2. **User `denied_commands`**: Explicitly blocked commands
3. **User `allowed_commands`**: Explicitly allowed commands
4. **`default_action`**: Fallback (default: `deny`)

### Environment Aliases

Environment aliases allow different secret key mappings per environment:

```yaml
env_aliases:
  dev:
    - pattern: "db/*"      # Match keys like db/host, db/password
      target: "dev/db/*"   # Transform to dev/db/host, dev/db/password
  prod:
    - pattern: "db/*"
      target: "prod/db/*"
```

**Usage:**

```bash
# CLI: Use --env flag
secretctl run --env=prod -k "db/*" -- ./deploy.sh

# MCP: Use env parameter
{
  "keys": ["db/*"],
  "command": "./deploy.sh",
  "env": "prod"
}
```

---

## Validation Limits

### Master Password

| Constraint | Value |
|------------|-------|
| Minimum length | 8 characters |
| Maximum length | 128 characters |

### Secret Keys

| Constraint | Value |
|------------|-------|
| Minimum length | 1 character |
| Maximum length | 256 characters |
| Allowed characters | Alphanumeric, `-`, `_`, `/`, `.` |

### Secret Values

| Constraint | Value |
|------------|-------|
| Maximum size | 1 MB (1,048,576 bytes) |

### Metadata

| Constraint | Value |
|------------|-------|
| Maximum notes size | 10 KB (10,240 bytes) |
| Maximum URL length | 2,048 characters |
| Maximum tag count | 10 tags |
| Maximum tag length | 64 characters |

---

## Unlock Cooldown

To protect against brute-force attacks, secretctl implements progressive cooldowns after failed unlock attempts:

| Failed Attempts | Cooldown Duration |
|----------------|-------------------|
| 5 | 30 seconds |
| 10 | 5 minutes |
| 20 | 30 minutes |

The cooldown counter resets after a successful unlock.

---

## Disk Space Requirements

secretctl monitors available disk space:

| Threshold | Action |
|-----------|--------|
| < 10 MB free | Vault operations blocked |
| > 90% used | Warning displayed |
| < 1 MB free | Audit logging blocked |

---

## Cryptographic Parameters

### Key Derivation (Argon2id)

| Parameter | Value |
|-----------|-------|
| Algorithm | Argon2id |
| Memory | 64 MB |
| Iterations | 3 |
| Parallelism | 4 threads |
| Salt length | 16 bytes (128-bit) |
| Output length | 32 bytes (256-bit) |

These parameters follow OWASP recommendations for high-security applications.

### Encryption (AES-256-GCM)

| Parameter | Value |
|-----------|-------|
| Algorithm | AES-256-GCM |
| Key length | 256 bits |
| Nonce length | 12 bytes (96-bit) |
| Tag length | 16 bytes (128-bit) |

### HMAC (Audit Chain)

| Parameter | Value |
|-----------|-------|
| Algorithm | HMAC-SHA256 |
| Key derivation | HKDF-SHA256 |
| Chain validation | Sequential verification |

---

## Shell Completion

Install shell completion for enhanced CLI experience:

### Bash

```bash
secretctl completion bash > /etc/bash_completion.d/secretctl
# or for user-level installation
secretctl completion bash > ~/.local/share/bash-completion/completions/secretctl
```

### Zsh

```bash
secretctl completion zsh > "${fpath[1]}/_secretctl"
# or specify a custom directory
secretctl completion zsh > ~/.zsh/completions/_secretctl
```

### Fish

```bash
secretctl completion fish > ~/.config/fish/completions/secretctl.fish
```

### PowerShell

```powershell
secretctl completion powershell | Out-String | Invoke-Expression
# or save to profile
secretctl completion powershell >> $PROFILE
```

---

## Claude Code Integration

Configure secretctl MCP server in `~/.claude.json`:

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

**Security Considerations:**

- Store `SECRETCTL_PASSWORD` securely (consider using a keychain or secret manager)
- The MCP server only exposes metadata and masked values to AI agents
- Configure `mcp-policy.yaml` to restrict which commands can be executed

See [MCP Integration Guide](/docs/guides/mcp/) for detailed setup instructions.
