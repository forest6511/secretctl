---
title: Security Model
description: How AI-Safe Access keeps secrets safe from AI agents.
sidebar_position: 2
---

# Security Model (AI-Safe Access)

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

## Threat Categories

secretctl's AI-Safe Access design protects against these threat categories:

### 1. Direct Secret Exposure

| Threat | Protection | Status |
|--------|------------|--------|
| AI requests plaintext secret | No `secret_get` tool exists | ✅ Mitigated |
| AI extracts from command output | Output sanitization | ✅ Mitigated |
| AI infers from partial data | Fixed-length masking (`****WXYZ`) | ✅ Mitigated |

### 2. Prompt Injection Attacks

| Threat | Protection | Status |
|--------|------------|--------|
| Malicious prompt requests secrets | Tool-level restrictions | ✅ Mitigated |
| Injected command in `secret_run` | Command allowlist policy | ✅ Mitigated |
| Encoded secret extraction | Not detected (see limitations) | ⚠️ Not Mitigated |

### 3. Command Execution Risks

| Threat | Protection | Status |
|--------|------------|--------|
| Environment variable dump | `env`/`printenv`/`set` blocked | ✅ Mitigated |
| Shell escape sequences | Command validation | ✅ Mitigated |
| Timeout/resource exhaustion | 300s timeout, resource limits | ✅ Mitigated |
| Arbitrary command execution | Deny-by-default policy | ✅ Mitigated |

### 4. Indirect Disclosure

| Threat | Protection | Status |
|--------|------------|--------|
| Timing attacks | Not applicable (local only) | N/A |
| Side-channel via output length | Fixed masking format | ✅ Mitigated |
| Model training data leakage | No plaintext to AI | ✅ Mitigated |

## Sanitization Details

Sanitization uses **exact string matching** to replace secret values in command output.

### What IS Detected

| Pattern | Example | Replacement |
|---------|---------|-------------|
| Exact match | `AKIAIOSFODNN7EXAMPLE` | `[REDACTED:aws/key]` |
| In JSON output | `{"key": "secret123"}` | `{"key": "[REDACTED:api/key]"}` |
| In URLs | `https://api.example.com?token=abc123` | `[REDACTED:api/token]` |

### What is NOT Detected

⚠️ **Known Limitations:**

| Pattern | Example | Why Not Detected |
|---------|---------|------------------|
| Base64 encoding | `QUtJQUlPU0ZPRE5ON0VYQU1QTEU=` | Only exact match |
| Hex encoding | `414b494149...` | Only exact match |
| URL encoding | `%41%4B%49%41...` | Only exact match |
| Partial matches | First 10 chars of secret | Only exact match |
| Case variations | `SECRET123` vs `secret123` | Case-sensitive match |
| Split output | `SEC` + `RET123` (across lines) | Single-pass detection |
| Compressed data | gzip/deflate encoded | Binary not scanned |

### Sanitization Timing

```
Command executes → stdout/stderr captured → Sanitization runs → Result to AI
                                                    ↑
                                           All secret values checked
```

**Important**: Sanitization happens **after** command completion. Secrets are exposed to the subprocess but never returned to the AI.

## Best Practices

1. **Use strong master password** - The MCP server requires your master password
2. **Limit allowed commands** - Only allow commands you actually need
3. **Review policy regularly** - Audit your allowed commands list
4. **Use key prefixes** - Organize secrets with prefixes (e.g., `aws/`, `db/`)
5. **Set expirations** - Use `--expires` when setting sensitive secrets
6. **Monitor audit logs** - Check `secretctl audit list` for unusual activity
7. **Avoid encoding secrets** - Don't store base64/hex encoded values as secrets
8. **Use short-lived tokens** - Prefer tokens with expiration over long-lived credentials
