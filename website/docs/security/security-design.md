---
title: Security Design
description: Comprehensive security architecture and design decisions.
sidebar_position: 5
---

# Security Design Document

This document provides a comprehensive overview of secretctl's security architecture, design decisions, and implementation details. It is intended for security-conscious users, auditors, and contributors.

## Executive Summary

secretctl implements a defense-in-depth security architecture with these core principles:

| Principle | Implementation |
|-----------|----------------|
| **AI-Safe Access** | AI agents never receive plaintext secrets |
| **Local-First** | No cloud dependency, all data stays on your machine |
| **Standard Crypto** | AES-256-GCM + Argon2id (OWASP compliant) |
| **Minimal Trust** | Go stdlib + golang.org/x/crypto only |
| **Full Auditability** | HMAC-chained tamper-evident logs |

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         secretctl SECURITY ARCHITECTURE                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        ACCESS LAYER                                  │   │
│  │                                                                      │   │
│  │   CLI               Desktop App            MCP Server                │   │
│  │   ────              ───────────            ──────────                │   │
│  │   Human Trust       Human Trust            AI Trust                  │   │
│  │   Full Access       Full Access            Restricted                │   │
│  │                                            (AI-Safe Access)          │   │
│  └──────────────────────────────┬──────────────────────────────────────┘   │
│                                 │                                           │
│  ┌──────────────────────────────┴──────────────────────────────────────┐   │
│  │                       POLICY LAYER                                   │   │
│  │                                                                      │   │
│  │   MCP Policy Engine        Command Validation       Rate Limiting    │   │
│  │   ─────────────────        ──────────────────       ─────────────    │   │
│  │   allowed_keys             blocked_commands         5 concurrent     │   │
│  │   denied_keys              timeout enforcement      max executions   │   │
│  │   capabilities             non-TTY mode                              │   │
│  └──────────────────────────────┬──────────────────────────────────────┘   │
│                                 │                                           │
│  ┌──────────────────────────────┴──────────────────────────────────────┐   │
│  │                       CRYPTO LAYER                                   │   │
│  │                                                                      │   │
│  │   Key Derivation           Encryption              Integrity         │   │
│  │   ──────────────           ──────────              ─────────         │   │
│  │   Argon2id                 AES-256-GCM             HMAC-SHA256       │   │
│  │   64MB memory              Random nonce            Chain verification│   │
│  │   3 iterations             Per-secret                                │   │
│  │   4 threads                                                          │   │
│  └──────────────────────────────┬──────────────────────────────────────┘   │
│                                 │                                           │
│  ┌──────────────────────────────┴──────────────────────────────────────┐   │
│  │                       STORAGE LAYER                                  │   │
│  │                                                                      │   │
│  │   vault.db (0600)          vault.salt (0600)       audit.log (0600)  │   │
│  │   ─────────────────        ────────────────        ────────────────  │   │
│  │   Encrypted secrets        128-bit random          HMAC chain        │   │
│  │   Encrypted DEK            Per-vault               Tamper detection  │   │
│  │   Metadata                                                           │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Key Hierarchy

secretctl uses a three-tier key hierarchy for defense-in-depth:

```
User Input: Master Password
           │
           ▼ Argon2id (m=64MB, t=3, p=4, salt=128-bit)
           │
    Master Key (256-bit, memory only)
           │
           ▼ AES-256-GCM encrypt/decrypt
           │
    Data Encryption Key (256-bit, stored encrypted)
           │
           ▼ AES-256-GCM encrypt/decrypt
           │
    Encrypted Secrets (stored in SQLite)
```

### Design Rationale

| Layer | Purpose | Security Property |
|-------|---------|-------------------|
| **Master Password** | User authentication | Never stored, not recoverable |
| **Master Key** | Protects DEK | Never persisted, derived on unlock |
| **DEK** | Encrypts secrets | Enables password rotation without re-encryption |
| **Encrypted Secrets** | Protected data | Confidentiality + integrity via GCM |

### Password Rotation Support

The DEK layer enables password changes without re-encrypting all secrets:

```
Old Password ─▶ Old Master Key ─▶ Decrypt DEK
                                       │
                                       ▼
New Password ─▶ New Master Key ─▶ Re-encrypt DEK
```

## Cryptographic Specifications

### Argon2id Parameters (OWASP 2025 Compliant)

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| Memory | 64 MB | Balance between security and usability |
| Iterations | 3 | Recommended for interactive use |
| Parallelism | 4 | Typical CPU core count |
| Salt | 128 bits | Unique per vault |
| Output | 256 bits | AES-256 key requirement |

**Why Argon2id?**
- Memory-hard: Resists GPU/ASIC attacks
- Hybrid: Combines Argon2i (side-channel resistant) and Argon2d (GPU resistant)
- Standard: Winner of Password Hashing Competition

### AES-256-GCM

| Property | Value |
|----------|-------|
| Key size | 256 bits |
| Nonce | 96 bits, random per encryption |
| Tag | 128 bits |
| Mode | GCM (authenticated encryption) |

**Why AES-256-GCM?**
- Authenticated: Provides both confidentiality and integrity
- Standard: NIST-approved, widely audited
- Performant: Hardware acceleration on modern CPUs

### HMAC Chain (Audit Logs)

```
Record N:
  hmac = HMAC-SHA256(
    id || op || key || timestamp || result || prev,
    audit_key
  )

audit_key = HKDF-SHA256(master_key, "audit-log-v1", 32)
```

## AI-Safe Access Design

### Core Principle

> AI agents use secrets without seeing them.

This follows 1Password's "Access Without Exposure" philosophy.

### Implementation

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          AI-SAFE ACCESS IMPLEMENTATION                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  PROHIBITED (not implemented)        PERMITTED (with restrictions)          │
│  ════════════════════════════        ════════════════════════════           │
│                                                                             │
│  secret_get (plaintext)              secret_list (names only)               │
│  secret_set (write)                  secret_exists (metadata)               │
│  secret_delete (destructive)         secret_get_masked (****WXYZ)           │
│  export (bulk access)                secret_run (env injection)             │
│                                      secret_list_fields (field names)       │
│                                      secret_get_field (non-sensitive)       │
│                                      secret_run_with_bindings               │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### secret_run Security Layers

```
AI Request: secret_run(keys=["aws/*"], command="aws s3 ls")
    │
    ▼
┌─────────────────────────────────────────────────────┐
│ Layer 1: Input Validation                            │
│   • Command blocklist (env, printenv, export, etc.) │
│   • Key pattern validation                          │
│   • Policy engine check                             │
└───────────────────────┬─────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│ Layer 2: Execution Isolation                         │
│   • Subprocess with clean environment               │
│   • Non-TTY mode enforced                           │
│   • Timeout: configurable, max 1 hour               │
│   • Temporary working directory                     │
└───────────────────────┬─────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│ Layer 3: Secret Injection                            │
│   • Secrets set as environment variables            │
│   • Values exist only in subprocess memory          │
│   • Never passed through AI context                 │
└───────────────────────┬─────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────┐
│ Layer 4: Output Sanitization                         │
│   • Scan stdout/stderr for secret values            │
│   • Replace matches with [REDACTED:key]             │
│   • Return sanitized flag in response               │
└───────────────────────┬─────────────────────────────┘
                        │
                        ▼
AI Response: {"exit_code": 0, "stdout": "bucket1\n", "sanitized": true}
```

### Blocked Commands

Commands that could expose environment variables are blocked:

```go
var defaultDeniedCommands = []string{
    "env",                  // Lists all environment variables
    "printenv",             // Prints environment variables
    "set",                  // Shell built-in showing env
    "export",               // Shell built-in for env
    "cat /proc/*/environ",  // Linux process environment
}
```

## Policy Engine

### Configuration

The policy engine controls which commands can be executed via `secret_run`:

```yaml
# ~/.secretctl/mcp-policy.yaml
version: 1
default_action: allow          # allow | deny
denied_commands:               # Always blocked (in addition to defaults)
  - rm
  - dd
allowed_commands:              # Explicitly permitted (when default_action: deny)
  - aws
  - kubectl
env_aliases:                   # Environment-specific key resolution
  dev:
    - pattern: "db/*"
      target: "db/dev/*"
  prod:
    - pattern: "db/*"
      target: "db/prod/*"
```

### Evaluation Order

1. **Default denied first**: Check built-in blocked commands (env, printenv, etc.)
2. **User denied commands**: Check `denied_commands` patterns
3. **Allowed commands**: If `default_action: deny`, check `allowed_commands`
4. **Default action**: Apply `default_action` if no explicit match
5. **Audit logging**: Record access attempt

:::info Key-Level Access Control
The current policy engine controls command execution, not key-level access. Field sensitivity (sensitive vs non-sensitive) is enforced at the MCP tool level via AI-Safe Access design. Future versions may add per-key access policies.
:::

## Audit System

### Log Storage

Audit logs are stored as monthly JSONL files with a metadata file:

```
~/.secretctl/
├── audit/
│   ├── audit.meta          # Chain state (sequence, prevHash)
│   ├── 2025-01.jsonl       # January 2025 events
│   └── 2025-02.jsonl       # February 2025 events
```

### Log Entry Structure

```json
{
  "v": 1,
  "id": "01HQ5E7N8KM...",
  "ts": "2025-01-15T10:30:00.123456789Z",
  "op": "run",
  "key": "aws/credentials",
  "key_hmac": "a3f2b1...",
  "actor": {
    "type": "user",
    "source": "mcp",
    "session_id": "..."
  },
  "result": "success",
  "ctx": {
    "command": "aws s3 ls",
    "exit_code": 0
  },
  "chain": {
    "seq": 42,
    "prev": "7c8d4e...",
    "hmac": "b4e5f6..."
  }
}
```

### Integrity Verification

```bash
$ secretctl audit verify
✓ 15,234 records verified
✓ Chain integrity: OK
✓ No gaps detected

# If tampering detected:
$ secretctl audit verify
✗ Chain broken at record id=01HQ5E7N8K...
  Expected prev: a3f2b1...
  Actual prev:   7c8d4e...
  ALERT: Possible tampering detected
```

## File Permissions

| File | Permission | Contents |
|------|------------|----------|
| `~/.secretctl/` | 0700 | Vault directory |
| `vault.db` | 0600 | Encrypted secrets, encrypted DEK, salt (in vault_keys table) |
| `vault.salt` | 0600 | Legacy Argon2 salt (128-bit), migrated to DB |
| `vault.meta` | 0600 | Version, timestamps |
| `audit/` | 0700 | Audit log directory |
| `audit/*.jsonl` | 0600 | Monthly HMAC-chained audit records |
| `audit/audit.meta` | 0600 | Chain state metadata |
| `mcp-policy.yaml` | 0600 | MCP access policies |

## Supply Chain Security

### Minimal Dependencies

secretctl uses only:
- Go standard library
- `golang.org/x/crypto` (Argon2, HKDF)
- `modernc.org/sqlite` (storage, Pure Go implementation)
- `github.com/spf13/cobra` (CLI framework)

### Verification

```bash
# Verify binary checksum
sha256sum secretctl-darwin-arm64

# Check dependencies
go list -m all
```

## Known Limitations

### Memory Protection

Go's garbage collector manages memory allocation. This means:
- Secrets may remain in memory until garbage collected
- Guaranteed secret zeroing is not possible

**Mitigation**: secretctl minimizes secret lifetime and scope.

### Out of Scope Threats

Consistent with industry standards (Vault, 1Password, Infisical):
- Root/kernel-level compromise
- Physical device access (unlocked)
- Weak master passwords
- Memory dump attacks

See [Threat Model](/docs/security/threat-model) for details.

## Version History

| Version | Security Changes |
|---------|------------------|
| v0.8.x | Multi-field secrets, field sensitivity, password change |
| v0.7.x | AI-Safe Access terminology, secret_get_masked enhancement |
| v0.6.x | Bindings support for secret_run |
| v0.5.x | Audit log HMAC chain, output sanitization |
| v0.4.x | MCP server, AI-Safe Access implementation |
| v0.3.x | Initial Argon2id + AES-256-GCM implementation |

## Related Documentation

- [Threat Model](/docs/security/threat-model) - Visual threat model
- [How It Works](/docs/security/how-it-works) - Technical architecture
- [Encryption Details](/docs/security/encryption) - Cryptographic specifications
- [MCP Tools Reference](/docs/reference/mcp-tools) - AI integration security

## Security Contact

Found a vulnerability? Please report responsibly:
- **Email**: secretctl.oss@gmail.com
- **Do NOT** open public GitHub issues
- See [SECURITY.md](https://github.com/forest6511/secretctl/blob/main/SECURITY.md)
